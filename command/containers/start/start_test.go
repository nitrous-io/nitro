package start_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/containers/start"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "containers/start")
}

var _ = Describe("containers/start", func() {
	var (
		cmd *start.Command
		ui  *cli.MockUi

		srv          *ghttp.Server
		handlers     http.HandlerFunc
		statusCode   int
		responseBody string
	)

	BeforeEach(func() {
		ui = &cli.MockUi{}

		handlers = ghttp.CombineHandlers(
			ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
			ghttp.VerifyRequest("POST", "/api/v0/containers/gregarious-scribe-24/start"),
			ghttp.RespondWithPtr(&statusCode, &responseBody),
		)
		srv = ghttp.NewServer()
		srv.AppendHandlers(handlers)

		cmd = &start.Command{
			UI:     ui,
			Server: srv.URL(),
			Client: httpclient.New("correct_token"),
		}

		command.Env.AuthToken = "a_token"
	})

	AfterEach(func() {
		srv.Close()
	})

	Context("nitro start containerslug", func() {
		doCmd := func() int {
			return cmd.Run([]string{"gregarious-scribe-24"})
		}

		Context("when the API responds with the container", func() {
			BeforeEach(func() {
				statusCode = http.StatusOK
				responseBody = `{ "container": { "slug": "gregarious-scribe-24", "image": "ubuntu", "state": "running" } }`
			})

			It("prints the container", func() {
				code := doCmd()
				Expect(ui.OutputWriter.String()).To(ContainSubstring(strings.TrimSpace(`
Slug                    Image     State
gregarious-scribe-24    ubuntu    running
`)))
				Expect(code).To(Equal(0))
			})
		})
	})

	Context("when not given any arguments", func() {
		It("prints usage and exits with error", func() {
			code := cmd.Run([]string{})
			Expect(ui.OutputWriter.String()).To(ContainSubstring("Usage:"))
			Expect(code).To(Equal(1))
		})
	})

	Context("when given > 1 argument", func() {
		It("prints usage and exits with error", func() {
			code := cmd.Run([]string{"gregarious-scribe-24", "hey", "man"})
			Expect(ui.OutputWriter.String()).To(ContainSubstring("Usage:"))
			Expect(code).To(Equal(1))
		})
	})
})
