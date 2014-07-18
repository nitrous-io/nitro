package create_test

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/containers/create"
	"github.com/nitrous-io/nitro/test"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "containers/create")
}

var _ = Describe("containers/create", func() {
	var (
		cmd        *create.Command
		ui         *cli.MockUi
		configPath string

		srv          *ghttp.Server
		handlers     http.HandlerFunc
		statusCode   int
		responseBody string
	)

	BeforeEach(func() {
		var err error

		handlers = ghttp.CombineHandlers(
			func(w http.ResponseWriter, req *http.Request) {
				// Call ParseForm() so that server.ReceivedRequests() have
				// POST params populated.
				err := req.ParseForm()
				Expect(err).To(BeNil())
				Expect(req.PostFormValue("host")).To(Equal("host-1"))
			},
			ghttp.VerifyHeaderKV("Authorization", "Bearer a_token"),
			ghttp.VerifyRequest("POST", "/api/v0/containers"),
			ghttp.RespondWithPtr(&statusCode, &responseBody),
		)
		srv = ghttp.NewServer()
		srv.AppendHandlers(handlers)

		ui = &cli.MockUi{}
		cmd = &create.Command{
			UI:     ui,
			Server: srv.URL(),
			Client: httpclient.New("a_token"),
		}
		configPath, err = test.WriteConfig(&command.ConfigFile{}, "")
		Expect(err).To(BeNil())

		command.Env.AuthToken = "a_token"
	})

	AfterEach(func() {
		srv.Close()
		os.Remove(configPath)
	})

	Context("when not all options are specified", func() {
		It("populates request with only given params", func() {
			cmd.Run([]string{
				"--host=host-1",
				"--slug=foo",
			})
			receivedRequests := srv.ReceivedRequests()
			Expect(receivedRequests).To(HaveLen(1))
			Expect(receivedRequests[0].PostFormValue("slug")).To(Equal("foo"))
		})
	})

	Context("when slug is not specified", func() {
		It("populates request with a randomly generated slug", func() {
			cmd.Run([]string{
				"--host=host-1",
			})
			receivedRequests := srv.ReceivedRequests()
			Expect(receivedRequests).To(HaveLen(1))
			Expect(receivedRequests[0].PostFormValue("slug")).NotTo(Equal(""))
		})
	})

	Context("when host is not specified", func() {
		doCmd := func() int {
			return cmd.Run([]string{
				"--slug=foo",
				"--image=ubuntu",
			})
		}
		Context("when there is no default host in the config file", func() {
			It("returns an error message without making a request", func() {
				code := doCmd()
				Expect(ui.ErrorWriter.String()).To(ContainSubstring("Please specify a host"))
				Expect(code).To(Equal(1))
			})
		})

		Context("when there is a default host in the config file", func() {
			BeforeEach(func() {
				_, err := test.WriteConfig(&command.ConfigFile{
					DefaultHost: "host-1",
				}, configPath)
				Expect(err).To(BeNil())

				// re-assign auth token
				command.Env.AuthToken = "a_token"
			})

			It("uses the default host", func() {
				statusCode = http.StatusCreated
				responseBody = `{"status":"created"}`

				code := doCmd()
				Expect(ui.OutputWriter.String()).To(ContainSubstring("Done"))
				Expect(code).To(Equal(0))
			})
		})
	})

	Context("when all options are provided", func() {
		doCmd := func() int {
			return cmd.Run([]string{
				"--host=host-1",
				"--slug=foo",
				"--image=ubuntu",
			})
		}

		It("populates request with all params", func() {
			doCmd()
			receivedRequests := srv.ReceivedRequests()
			Expect(receivedRequests).To(HaveLen(1))
			Expect(receivedRequests[0].PostFormValue("slug")).To(Equal("foo"))
			Expect(receivedRequests[0].PostFormValue("image")).To(Equal("ubuntu"))
		})

		Context("when request succeeds", func() {
			BeforeEach(func() {
				statusCode = http.StatusCreated
				responseBody = `{
					"status": "created",
					"container": { "slug": "wild-rocket-23", "image": "ubuntu", "state": "running" }
				}`
			})

			It("reports the container's slug", func() {
				code := doCmd()
				Expect(ui.OutputWriter.String()).To(ContainSubstring("wild-rocket-23"))
				Expect(code).To(Equal(0))
			})

			It("writes the host to the config file", func() {
				doCmd()

				configFile, err := command.LoadConfig()
				Expect(err).To(BeNil())
				Expect(configFile.DefaultHost).To(Equal("host-1"))
			})
		})

		Context("when request fails", func() {
			BeforeEach(func() {
				statusCode = http.StatusBadRequest
				responseBody = `{"error":"validation_failed","error_description":"image required"}`
			})

			It("displays error message", func() {
				code := doCmd()
				Expect(ui.ErrorWriter.String()).To(ContainSubstring("image required"))
				Expect(code).To(Equal(1))
			})

			It("doesn't write the host to the config file", func() {
				doCmd()

				f, err := os.Open(configPath)
				Expect(err).To(BeNil())
				var configFile command.ConfigFile
				err = json.NewDecoder(f).Decode(&configFile)
				Expect(err).To(BeNil())
				Expect(configFile.DefaultHost).To(Equal(""))
			})
		})
	})
})
