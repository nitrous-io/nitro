package stop_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/containers/stop"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "containers/stop")
}

var _ = Describe("containers/stop", func() {
	var (
		cmd *stop.Command
		ui  *command.MockUI
		r   *io.PipeReader
		w   *io.PipeWriter

		srv          *ghttp.Server
		handlers     http.HandlerFunc
		statusCode   int
		responseBody string
	)

	BeforeEach(func() {
		r, w = io.Pipe()
		ui = &command.MockUI{&cli.MockUi{InputReader: r}}

		handlers = ghttp.CombineHandlers(
			ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
			ghttp.VerifyRequest("POST", "/api/v0/containers/gregarious-scribe-24/stop"),
			ghttp.RespondWithPtr(&statusCode, &responseBody),
		)
		srv = ghttp.NewServer()
		srv.AppendHandlers(handlers)

		cmd = &stop.Command{
			UI:     ui,
			Server: srv.URL(),
			Client: httpclient.New("correct_token"),
		}

		command.Env.AuthToken = "a_token"
	})

	AfterEach(func() {
		r.Close()
		w.Close()
		srv.Close()
	})

	Context("nitro stop containerslug", func() {
		doCmd := func() int {
			return cmd.Run([]string{"gregarious-scribe-24", "--force"})
		}

		Context("when the API responds with the container", func() {
			BeforeEach(func() {
				statusCode = http.StatusOK
				responseBody = `{ "container": { "slug": "gregarious-scribe-24", "image": "ubuntu", "state": "stopped" } }`
			})

			It("prints the container", func() {
				code := doCmd()
				Expect(ui.OutputWriter.String()).To(ContainSubstring(strings.TrimSpace(`
Slug                    Image     State
gregarious-scribe-24    ubuntu    stopped
`)))
				Expect(code).To(Equal(0))
			})

			Context("when --force option is not specified", func() {
				doCmd := func() int {
					return cmd.Run([]string{"gregarious-scribe-24"})
				}

				It("asks for confirmation before proceeding", func() {
					go w.Write([]byte("n\n"))
					code := doCmd()
					Expect(ui.OutputWriter.String()).To(ContainSubstring("Cancelled"))
					Expect(code).To(Equal(1))

					go w.Write([]byte("y\n"))
					code = doCmd()
					Expect(ui.OutputWriter.String()).To(ContainSubstring("gregarious-scribe-24"))
					Expect(code).To(Equal(0))
				})
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
})
