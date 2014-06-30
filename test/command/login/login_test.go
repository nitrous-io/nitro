package logintest

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/mitchellh/cli"
	. "github.com/nitrous-io/nitro/command/login"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func authTestServer(username, password string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		u := r.Form.Get("username")
		p := r.Form.Get("password")

		type message struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
			AccessToken      string `json:"access_token"`
		}
		var m message
		if username == u && password == p {
			m = message{AccessToken: "abcd1234"}
		} else {
			m = message{Error: "invalid_login", ErrorDescription: "Cannot login with the provided credentials"}
			w.WriteHeader(401)
		}
		payload, err := json.Marshal(m)
		if err != nil {
			panic(err)
		}

		w.Write(payload)
	}))
}

var _ = Describe("LoginCommand", func() {
	var (
		cmd *LoginCommand
		ui  *cli.MockUi

		username   string
		password   string
		args       []string
		testServer *httptest.Server
	)

	BeforeEach(func() {
		username = "iamclovin"
		password = "man u 4evar!!1!"

		testServer = authTestServer(username, password)

		ui = &cli.MockUi{}
		cmd = &LoginCommand{
			UI:            ui,
			TokenFilePath: "/tmp/.nitrotoken",
			Server:        testServer.URL,
		}
	})

	AfterEach(func() {
		testServer.Close()
	})

	It("implements cli.Command", func() {
		var _ cli.Command = cmd
	})

	Context("with a valid username and password", func() {
		BeforeEach(func() {
			args = []string{
				"-u", username,
				"-p", password,
			}

			exitCode := cmd.Run(args)
			Expect(exitCode).To(Equal(0))
		})

		AfterEach(func() {
			err := os.Remove("/tmp/.nitrotoken")
			if err != nil {
				panic(err)
			}
		})

		It("writes the access token to .nitrotoken file", func() {
			token, err := ioutil.ReadFile("/tmp/.nitrotoken")
			Expect(string(token[:])).To(Equal("abcd1234"))
			Expect(err).To(BeNil())
		})

		It("prints a message saying logged in as the given username", func() {
			Expect(ui.OutputWriter.String()).To(ContainSubstring("Logged in as '%s'", username))
		})
	})

	Context("with an invalid username and password", func() {
		BeforeEach(func() {
			args = []string{
				"-u", "foo",
				"-p", "bar",
			}

			exitCode := cmd.Run(args)
			Expect(exitCode).To(Equal(1))
		})

		It("doesnt write the access token to .nitrotoken file", func() {
			_, err := ioutil.ReadFile("/tmp/.nitrotoken")
			Expect(err).NotTo(BeNil())
		})

		It("prints a message saying failed to authenticate", func() {
			Expect(ui.ErrorWriter.String()).To(ContainSubstring("Failed to login"))
		})
	})
})
