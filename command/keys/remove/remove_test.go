package remove_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/keys/remove"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "keys/remove")
}

var _ = Describe("keys/remove", func() {
	var (
		cmd          *remove.Command
		ui           *command.MockUI
		r            *io.PipeReader
		w            *io.PipeWriter
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
			ghttp.VerifyRequest("DELETE", "/api/v0/containers/gregarious-scribe-24/keys/fprint"),
			ghttp.RespondWithPtr(&statusCode, &responseBody),
		)
		srv = ghttp.NewServer()
		srv.AppendHandlers(handlers)

		cmd = &remove.Command{
			UI:     ui,
			Server: srv.URL(),
			Client: httpclient.New("correct_token"),
		}
	})

	AfterEach(func() {
		// Reset it after every test so that any error conditions that
		// are set in contexts don't linger
		statusCode = http.StatusOK
		responseBody = `{"keys":[],"status":"ok"}`

		r.Close()
		w.Close()
		srv.Close()
	})

	Context("nitro keys:remove containerslug fingerprint", func() {
		doCmd := func() int {
			return cmd.Run([]string{"gregarious-scribe-24", "fprint", "--yes"})
		}

		Context("successful removal of key", func() {
			BeforeEach(func() {
				statusCode = http.StatusOK
				responseBody = `{"status": "ok"}`
			})

			It("outputs success msg", func() {
				code := doCmd()
				Expect(ui.OutputWriter.String()).To(ContainSubstring("Removed key"))
				Expect(code).To(Equal(0))
			})
		})

		Context("JSON response with error", func() {
			BeforeEach(func() {
				statusCode = http.StatusBadRequest
				responseBody = `{"error": "validation_failed", "error_description": "container not found"}`
			})

			It("outputs error message", func() {
				code := doCmd()
				Expect(ui.ErrorWriter.String()).To(Equal("container not found\n"))
				Expect(code).To(Equal(1))
			})
		})

		Context("when --yes option is not specified", func() {
			doCmd := func() int {
				return cmd.Run([]string{"gregarious-scribe-24", "fprint"})
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
				Expect(ui.OutputWriter.String()).To(ContainSubstring("gregarious-scribe-24"))
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
})
