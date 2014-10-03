package remove_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/images/remove"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "images/remove")
}

var _ = Describe("images/remove", func() {
	var (
		cmd          *remove.Command
		ui           *command.MockUI
		r            *io.PipeReader
		w            *io.PipeWriter
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
			ghttp.VerifyRequest("DELETE", "/api/v0/hosts/myhost/images/nitrousio/ubuntu"),
			ghttp.RespondWithPtr(&statusCode, &responseBody),
		))

		cmd = &remove.Command{
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

	Context("nitro images:remove host image", func() {
		doCmd := func() int {
			return cmd.Run([]string{"myhost", "nitrousio/ubuntu", "--yes"})
		}

		Context("successful removal of image", func() {
			BeforeEach(func() {
				statusCode = http.StatusOK
				responseBody = `{"status": "ok"}`
			})

			It("outputs success msg", func() {
				code := doCmd()
				Expect(ui.OutputWriter.String()).To(ContainSubstring("Removed image 'nitrousio/ubuntu' from host 'myhost'"))
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

		Context("when --yes option is not specified", func() {
			doCmd := func() int {
				return cmd.Run([]string{"myhost", "nitrousio/ubuntu"})
			}

			BeforeEach(func() {
				statusCode = http.StatusOK
				responseBody = `{"status":"ok"}`
			})

			It("asks for confirmation before proceeding", func() {
				go w.Write([]byte("n\n"))
				code := doCmd()
				Expect(ui.OutputWriter.String()).To(ContainSubstring("Cancelled"))
				Expect(code).To(Equal(1))

				go w.Write([]byte("y\n"))
				code = doCmd()
				Expect(ui.OutputWriter.String()).To(ContainSubstring("myhost"))
				Expect(code).To(Equal(0))
			})
		})
	})

	Context("when given wrong arguments", func() {
		It("prints usage and exits with error", func() {
			code := cmd.Run([]string{})
			Expect(ui.OutputWriter.String()).To(ContainSubstring("Usage:"))
			Expect(code).To(Equal(1))
		})
	})

	Context("Help", func() {
		It("outputs usage", func() {
			Expect(cmd.Help()).To(ContainSubstring("Usage: nitro images:remove"))
		})
	})
})
