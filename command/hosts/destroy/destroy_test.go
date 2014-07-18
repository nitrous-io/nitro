package destroy_test

import (
	"net/http"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/hosts/destroy"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "hosts/destroy")
}

var _ = Describe("hosts/destroy", func() {
	var (
		cmd      *destroy.Command
		ui       *cli.MockUi
		srv      *ghttp.Server
		handlers []http.HandlerFunc
	)

	BeforeEach(func() {
		command.Env.AuthToken = "a_token"

		srv = ghttp.NewServer()
		ui = &cli.MockUi{}
		cmd = &destroy.Command{
			UI:     ui,
			Server: srv.URL(),
			Client: httpclient.New("a_token"),
		}
		handlers = []http.HandlerFunc{
			ghttp.VerifyHeader(http.Header{
				"Authorization": []string{"Bearer a_token"},
			}),
			ghttp.VerifyRequest("DELETE", "/api/v0/hosts/foo"),
		}

		command.Env.AuthToken = "a_token"
	})

	It("requires name as an argument", func() {
		code := cmd.Run([]string{})
		Expect(code).To(Equal(1))
		Expect(ui.OutputWriter.String()).To(ContainSubstring("Usage:"))
	})

	Context("hitting the server", func() {
		Context("when request succeeds", func() {
			It("reports success", func() {
				handlers = append(handlers,
					ghttp.RespondWith(http.StatusOK, `{"status":"ok"}`))
				srv.AppendHandlers(ghttp.CombineHandlers(handlers...))

				code := cmd.Run([]string{"foo"})

				Expect(ui.OutputWriter.String()).To(ContainSubstring("Done"))
				Expect(code).To(Equal(0))
			})
		})

		Context("when request fails", func() {
			It("displays error message", func() {
				handlers = append(handlers,
					ghttp.RespondWith(http.StatusBadRequest,
						`{"error":"invalid_request","error_description":"host not found"}`),
				)
				srv.AppendHandlers(ghttp.CombineHandlers(handlers...))

				code := cmd.Run([]string{"foo"})

				Expect(ui.ErrorWriter.String()).To(ContainSubstring("host not found"))
				Expect(code).To(Equal(1))
			})
		})
	})
})
