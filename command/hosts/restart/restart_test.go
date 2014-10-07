package restart_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/hosts/restart"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "hosts/restart")
}

var _ = Describe("hosts/restart", func() {
	var (
		cmd *restart.Command
		ui  *command.MockUI
		r   *io.PipeReader
		w   *io.PipeWriter

		srv          *ghttp.Server
		statusCode   int
		responseBody string
	)

	BeforeEach(func() {
		r, w = io.Pipe()
		ui = &command.MockUI{&cli.MockUi{InputReader: r}}

		srv = ghttp.NewServer()
		srv.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
			ghttp.VerifyRequest("POST", "/api/v0/hosts/foo-host-42/restart"),
			func(w http.ResponseWriter, req *http.Request) {
				req.ParseForm()
			},
			ghttp.RespondWithPtr(&statusCode, &responseBody),
		))

		cmd = &restart.Command{
			UI:     ui,
			Server: srv.URL(),
			Client: httpclient.New("correct_token"),
		}
	})

	AfterEach(func() {
		r.Close()
		w.Close()
		srv.Close()
	})

	Context("nitro hosts:restart slug", func() {
		doCmd := func() int {
			return cmd.Run([]string{"foo-host-42", "--yes"})
		}

		Context("when host is successfully restarted", func() {
			BeforeEach(func() {
				statusCode = http.StatusOK
				responseBody = `{"status":"ok"}`
			})

			It("reports success", func() {
				code := doCmd()
				Expect(ui.OutputWriter.String()).To(ContainSubstring("Done"))
				Expect(code).To(Equal(0))
			})

			Context("when --yes option is not specified", func() {
				doCmd := func() int {
					return cmd.Run([]string{"foo-host-42"})
				}

				It("asks for confirmation before proceeding", func() {
					go w.Write([]byte("n\n"))
					code := doCmd()
					Expect(ui.OutputWriter.String()).To(ContainSubstring("Cancelled"))
					Expect(code).To(Equal(1))

					go w.Write([]byte("y\n"))
					code = doCmd()
					Expect(ui.OutputWriter.String()).To(ContainSubstring("Done"))
					Expect(code).To(Equal(0))
				})
			})

			Context("when --force option is given", func() {
				doCmd := func() int {
					return cmd.Run([]string{"foo-host-42", "--yes", "--force"})
				}

				It("makes request with param force=true", func() {
					code := doCmd()
					Expect(code).To(Equal(0))

					reqs := srv.ReceivedRequests()
					Expect(reqs[0].PostFormValue("force")).To(Equal("true"))
				})
			})
		})

		Context("when host cannot be restarted", func() {
			BeforeEach(func() {
				statusCode = http.StatusBadRequest
				responseBody = `{"error":"invalid_request","error_description":"host not found"}`
			})

			It("displays error message", func() {
				code := doCmd()
				Expect(ui.ErrorWriter.String()).To(ContainSubstring("host not found"))
				Expect(code).To(Equal(1))
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
