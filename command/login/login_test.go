package login_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/auth"
	"github.com/nitrous-io/nitro/command"
	. "github.com/nitrous-io/nitro/command/login"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLogin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "login")
}

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
			m = message{AccessToken: "access_token"}
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

		testConfigFile *os.File
	)

	BeforeEach(func() {
		username = "iamclovin"
		password = "man u 4evar!!1!"

		testServer = authTestServer(username, password)

		ui = &cli.MockUi{}
		cmd = &LoginCommand{
			UI:     ui,
			Server: testServer.URL,
		}

		f, err := ioutil.TempFile("", "nitrocfg")
		Expect(err).To(BeNil())
		testConfigFile = f
		command.Env.ConfigPath = f.Name()
	})

	AfterEach(func() {
		testServer.Close()
		testConfigFile.Close()
		os.Remove(testConfigFile.Name())
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
			Expect(ui.ErrorWriter.String()).To(BeEmpty())
			Expect(exitCode).To(Equal(0))
		})

		It("adds a new token to the auth config file", func() {
			c, err := auth.LoadConfig(testConfigFile.Name())
			Expect(err).To(BeNil())
			Expect(c.Configs[testServer.URL].Username).To(Equal(username))
			Expect(c.Configs[testServer.URL].Token).To(Equal("access_token"))
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

		It("does not add a new token to the auth config file", func() {
			c, err := auth.LoadConfig(testConfigFile.Name())
			Expect(err).To(BeNil())
			Expect(c.Configs[testServer.URL].Username).To(BeEmpty())
			Expect(c.Configs[testServer.URL].Token).To(BeEmpty())
		})

		It("prints a message saying failed to authenticate", func() {
			Expect(ui.ErrorWriter.String()).To(ContainSubstring("Failed to login"))
		})
	})
})
