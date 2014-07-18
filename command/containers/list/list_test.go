package list_test

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/containers/list"
	"github.com/nitrous-io/nitro/test"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "containers/list")
}

var _ = Describe("containers/list", func() {
	var (
		cmd        *list.Command
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
			ghttp.VerifyHeaderKV("Authorization", "Bearer a_token"),
			ghttp.VerifyRequest("GET", "/api/v0/containers", "host=host-1"),
			ghttp.RespondWithPtr(&statusCode, &responseBody),
		)
		srv = ghttp.NewServer()
		srv.AppendHandlers(handlers)

		ui = &cli.MockUi{}
		cmd = &list.Command{
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

	Context("when host is not specified", func() {
		Context("when there is no default host in the config file", func() {
			It("prints usage and exits with error", func() {
				code := cmd.Run([]string{})
				Expect(ui.OutputWriter.String()).To(ContainSubstring("Usage:"))
				Expect(code).To(Equal(1))
			})
		})

		Context("when there is a default host in the config file", func() {
			BeforeEach(func() {
				_, err := test.WriteConfig(&command.ConfigFile{
					DefaultHost: "host-1",
				}, configPath)
				Expect(err).To(BeNil())
				command.Env.AuthToken = "a_token"
			})

			It("uses the default host", func() {
				statusCode = http.StatusOK
				responseBody = `{"containers":[],"status":"ok"}`

				code := cmd.Run([]string{})
				Expect(code).To(Equal(0))
			})
		})
	})

	Context("when server responds with containers in host", func() {
		var (
			formattedOutput = strings.TrimSpace(`
Name   Image            State
foo    freighter.foo    running
`) + "\n\n"
		)

		BeforeEach(func() {
			statusCode = http.StatusOK
			responseBody = `{
				"containers": [
					{
						"id":"df5b37192a6469652b68639f5736973cf0fe9ec0fe4747ade3747d55548ccfdb",
						"user_id":1,
						"host_id":25,
						"image":"freighter.foo",
						"slug":"foo",
						"name":{"String":"foo","Valid":true},
						"ip_address":{"String":"10.1.0.2","Valid":true},
						"terminated_at":null,
						"touched_at":null,
						"created_at":"2014-07-15T05:44:09.037793Z",
						"updated_at":"2014-07-15T05:44:09.037793Z",
						"state":"running"
					}
				]
			}`
		})

		It("outputs formatted containers on success", func() {
			code := cmd.Run([]string{"host-1"})
			Expect(ui.OutputWriter.String()).To(ContainSubstring(formattedOutput))
			Expect(code).To(Equal(0))
		})

		It("writes the host to the config file", func() {
			cmd.Run([]string{"host-1"})

			configFile, err := command.LoadConfig()
			Expect(err).To(BeNil())
			Expect(configFile.DefaultHost).To(Equal("host-1"))
		})
	})

	Context("when host has no containers", func() {
		BeforeEach(func() {
			statusCode = http.StatusOK
			responseBody = `{"containers":[],"status":"ok"}`
		})

		It("displays no containers found", func() {
			code := cmd.Run([]string{"host-1"})
			Expect(ui.OutputWriter.String()).To(ContainSubstring("No containers found"))
			Expect(code).To(Equal(0))
		})

		It("writes the host to the config file", func() {
			cmd.Run([]string{"host-1"})

			f, err := os.Open(configPath)
			Expect(err).To(BeNil())
			var configFile command.ConfigFile
			err = json.NewDecoder(f).Decode(&configFile)
			Expect(err).To(BeNil())
			Expect(configFile.DefaultHost).To(Equal("host-1"))
		})
	})

	Context("when server responds with an internal_server_error", func() {
		BeforeEach(func() {
			statusCode = http.StatusInternalServerError
			responseBody = `{"error":"internal_server_error","error_hash":"b482fd5b602650c798366fe9ee8e6cf65ef759fc"}`
		})

		It("displays unknown error message", func() {
			code := cmd.Run([]string{"host-1"})
			Expect(ui.ErrorWriter.String()).To(ContainSubstring("Unknown error occurred"))
			Expect(code).To(Equal(1))
		})

		It("doesn't write the host to the config file", func() {
			cmd.Run([]string{"host-1"})

			f, err := os.Open(configPath)
			Expect(err).To(BeNil())
			var configFile command.ConfigFile
			err = json.NewDecoder(f).Decode(&configFile)
			Expect(err).To(BeNil())
			Expect(configFile.DefaultHost).To(Equal(""))
		})
	})
})
