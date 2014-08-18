package tag_test

import (
	"net/http"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/images/tag"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "images/tag")
}

var _ = Describe("images/tag", func() {
	var (
		cmd          *tag.Command
		ui           *command.MockUI
		srv          *ghttp.Server
		statusCode   int
		responseBody string
	)

	BeforeEach(func() {
		ui = &command.MockUI{&cli.MockUi{}}
		srv = ghttp.NewServer()
		srv.AppendHandlers(ghttp.CombineHandlers(
			func(w http.ResponseWriter, req *http.Request) {
				err := req.ParseForm()
				Expect(err).To(BeNil())
				Expect(req.PostFormValue("image")).To(Equal("scratch"))
			},
			ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
			ghttp.VerifyRequest("POST", "/api/v0/hosts/myhost/images/tag"),
			ghttp.RespondWithPtr(&statusCode, &responseBody),
		))

		cmd = &tag.Command{
			UI:     ui,
			Server: srv.URL(),
			Client: httpclient.New("correct_token"),
		}
	})

	AfterEach(func() {
		srv.Close()
	})

	Context("nitro images:tag host image", func() {
		doCmd := func() int {
			return cmd.Run([]string{"myhost", "scratch", "itch"})
		}

		Context("successful tag", func() {
			BeforeEach(func() {
				statusCode = http.StatusOK
				responseBody = `{"status":"OK"}`
			})

			It("outputs success message", func() {
				code := doCmd()
				Expect(ui.OutputWriter.String()).To(ContainSubstring("Tagged image"))
				Expect(code).To(Equal(0))
			})
		})

		Context("JSON response with error", func() {
			BeforeEach(func() {
				statusCode = http.StatusBadRequest
				responseBody = `{"error": "validation_failed", "error_description": "image not found"}`
			})

			It("outputs error message", func() {
				code := doCmd()
				Expect(ui.ErrorWriter.String()).To(Equal("image not found\n"))
				Expect(code).To(Equal(1))
			})
		})
	})

	Context("when given wrong arguments", func() {
		It("prints usage and exits with error", func() {
			for _, t := range [][]string{{}, {"hostonly"}, {"host", "image"}} {
				code := cmd.Run(t)
				Expect(ui.OutputWriter.String()).To(ContainSubstring("Usage:"))
				Expect(code).To(Equal(1))
			}
		})
	})

	Context("Help", func() {
		It("outputs usage", func() {
			Expect(cmd.Help()).To(ContainSubstring("Usage: nitro images:tag"))
		})
	})
})
