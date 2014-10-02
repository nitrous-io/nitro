package reconfig_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command/containers/reconfig"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "containers/reconfig")
}

var _ = Describe("containers/reconfig", func() {
	var (
		cmd *reconfig.Command
		ui  *cli.MockUi

		srv          *ghttp.Server
		handlers     http.HandlerFunc
		statusCode   int
		responseBody string

		expectedCmd string
	)

	BeforeEach(func() {
		handlers = ghttp.CombineHandlers(
			func(w http.ResponseWriter, req *http.Request) {
				err := req.ParseForm()
				Expect(err).To(BeNil())
				Expect(req.PostForm["volume[]"]).To(Equal([]string{"/shared", "/music"}))
				Expect(req.PostForm["port[]"]).To(Equal([]string{"8080:3000", "127.0.0.1:4242"}))
				if expectedCmd == "" {
					_, found := req.Form["cmd"]
					Expect(found).To(BeFalse())
				} else {
					Expect(req.PostFormValue("cmd")).To(Equal(expectedCmd))
				}
			},
			ghttp.VerifyHeaderKV("Authorization", "Bearer a_token"),
			ghttp.VerifyRequest("POST", "/api/v0/containers/wild-rocket-23/reconfig"),
			ghttp.RespondWithPtr(&statusCode, &responseBody),
		)
		srv = ghttp.NewServer()
		srv.AppendHandlers(handlers)

		ui = &cli.MockUi{}
		cmd = &reconfig.Command{
			UI:     ui,
			Server: srv.URL(),
			Client: httpclient.New("a_token"),
		}
	})

	AfterEach(func() {
		srv.Close()
	})

	Context("nitro reconfig containerslug", func() {
		BeforeEach(func() {
			expectedCmd = "/bin/dash -c \"echo 'hi'\""
		})

		doCmd := func() int {
			return cmd.Run([]string{
				"--volume=/shared",
				"--volume=/music",
				"--port-forward=8080:3000,127.0.0.1:4242",
				"--command=/bin/dash -c \"echo 'hi'\"",
				"wild-rocket-23",
			})
		}

		Context("when the API responds with the reconfigured container", func() {
			BeforeEach(func() {
				statusCode = http.StatusOK
				responseBody = `{ "container": { "slug": "wild-rocket-23", "image": "wild-rocket-reconfig-image", "state": "running" } }`
			})

			It("prints the container", func() {
				code := doCmd()
				Expect(ui.OutputWriter.String()).To(ContainSubstring(strings.TrimSpace(`
Slug              Image                         State
wild-rocket-23    wild-rocket-reconfig-image    running
`)))
				Expect(code).To(Equal(0))
			})
		})

		Context("when the API responds with an error", func() {
			BeforeEach(func() {
				statusCode = http.StatusBadRequest
				responseBody = `{"error":"invalid_request","error_description":"reconfig failed"}`
			})

			It("displays error message", func() {
				code := doCmd()
				Expect(ui.ErrorWriter.String()).To(ContainSubstring("Failed to reconfigure the container"))
				Expect(code).To(Equal(1))
			})
		})

		Context("when the command is not given", func() {
			BeforeEach(func() {
				expectedCmd = ""
			})
			It("should not send a cmd form value", func() {
				cmd.Run([]string{
					"--volume=/shared",
					"--volume=/music",
					"--port-forward=8080:3000,127.0.0.1:4242",
					"wild-rocket-23",
				})
			})
		})

	})

	Context("when slug is not given", func() {
		It("prints usage and exits with error", func() {
			code := cmd.Run([]string{})
			Expect(ui.OutputWriter.String()).To(ContainSubstring("Usage:"))
			Expect(code).To(Equal(1))
		})
	})
})
