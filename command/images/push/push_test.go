package push_test

import (
	"net/http"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/images/push"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "images/push")
}

var _ = Describe("images/push", func() {
	var (
		cmd          *push.Command
		ui           *command.MockUI
		srv          *ghttp.Server
		statusCode   int
		responseBody string
	)

	BeforeEach(func() {
		ui = &command.MockUI{&cli.MockUi{}}
		handlers := ghttp.CombineHandlers(
			func(w http.ResponseWriter, req *http.Request) {
				Expect(req.FormValue("repository")).To(Equal("username/myscratch"))
			},
			ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
			ghttp.VerifyRequest("POST", "/api/v0/hosts/myhost/images/scratch/push"),
			ghttp.RespondWithPtr(&statusCode, &responseBody),
		)
		srv = ghttp.NewServer()
		srv.AppendHandlers(handlers)

		cmd = &push.Command{
			UI:     ui,
			Server: srv.URL(),
			Client: httpclient.New("correct_token"),
		}
	})

	AfterEach(func() {
		srv.Close()
	})

	Context("nitro images:push host image", func() {
		doCmd := func() int {
			return cmd.Run([]string{"myhost", "scratch", "username/myscratch"})
		}

		BeforeEach(func() {
			statusCode = http.StatusOK
			responseBody = `{"status":"OK"}`
		})

		It("prints success message", func() {
			code := doCmd()
			Expect(ui.OutputWriter.String()).To(ContainSubstring("Pushed image"))
			Expect(code).To(Equal(0))
		})

		Context("when request fails with a HTTP 401 Unauthorized", func() {
			BeforeEach(func() {
				statusCode = http.StatusUnauthorized
				responseBody = `{"error": "invalid_request", "error_description": "don't care"}`
			})

			It("prints failure and a hint to login", func() {
				code := doCmd()
				Expect(ui.ErrorWriter.String()).To(ContainSubstring("Failed: "))
				Expect(ui.ErrorWriter.String()).To(ContainSubstring("nitro registry:login"))
				Expect(code).To(Equal(1))
			})
		})

		Context("when request fails", func() {
			BeforeEach(func() {
				statusCode = http.StatusBadRequest
				responseBody = `{"error": "validation_failed", "error_description": "image not found"}`
			})

			It("prints its error message", func() {
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
		It("prints usage", func() {
			Expect(cmd.Help()).To(ContainSubstring("Usage: nitro images:push"))
		})
	})
})
