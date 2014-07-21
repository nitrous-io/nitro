package destroy_test

import (
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/hosts/destroy"
	"github.com/nitrous-io/nitro/test"
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
		cmd          *destroy.Command
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
			ghttp.VerifyHeaderKV("Authorization", "Bearer a_token"),
			ghttp.VerifyRequest("DELETE", "/api/v0/hosts/rocky-host-12"),
			ghttp.RespondWithPtr(&statusCode, &responseBody),
		)
		srv = ghttp.NewServer()
		srv.AppendHandlers(handlers)

		cmd = &destroy.Command{
			UI:     ui,
			Server: srv.URL(),
			Client: httpclient.New("a_token"),
		}
	})

	AfterEach(func() {
		r.Close()
		w.Close()
		srv.Close()
	})

	It("requires name as an argument", func() {
		code := cmd.Run([]string{})
		Expect(code).To(Equal(1))
		Expect(ui.OutputWriter.String()).To(ContainSubstring("Usage:"))
	})

	Context("when host is successfully destroyed", func() {
		doCmd := func() int {
			return cmd.Run([]string{"rocky-host-12", "--force"})
		}

		BeforeEach(func() {
			statusCode = http.StatusOK
			responseBody = `{"status":"ok"}`
		})

		It("reports success", func() {
			code := doCmd()
			Expect(ui.OutputWriter.String()).To(ContainSubstring("Done"))
			Expect(code).To(Equal(0))
		})

		Context("when host is the default host in the config file", func() {
			It("unsets default host in config file", func() {
				configPath, err := test.WriteConfig(&command.ConfigFile{DefaultHost: "rocky-host-12"}, "")
				Expect(err).To(BeNil())
				defer os.Remove(configPath)

				doCmd()

				c, err := command.LoadConfig()
				Expect(err).To(BeNil())
				Expect(c.DefaultHost).To(Equal(""))
			})
		})

		Context("when host is not the default host in the config file", func() {
			It("doesn't change the default host in config file", func() {
				configPath, err := test.WriteConfig(&command.ConfigFile{DefaultHost: "another-host"}, "")
				Expect(err).To(BeNil())
				defer os.Remove(configPath)

				doCmd()

				c, err := command.LoadConfig()
				Expect(err).To(BeNil())
				Expect(c.DefaultHost).To(Equal("another-host"))
			})
		})
	})

	Context("when host cannot be destroyed", func() {
		doCmd := func() int {
			return cmd.Run([]string{"rocky-host-12", "--force"})
		}

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

	Context("attempting without the --force option", func() {
		doCmd := func() int {
			return cmd.Run([]string{"rocky-host-12"})
		}

		It("asks for confirmation and user cancels", func() {
			go w.Write([]byte("n\n"))
			code := doCmd()
			Expect(ui.OutputWriter.String()).To(ContainSubstring("Cancelled"))
			Expect(code).To(Equal(1))
		})

		It("asks for confirmation and user confirms", func() {
			statusCode = http.StatusOK
			responseBody = `{"status":"ok"}`

			go w.Write([]byte("y\n"))
			code := doCmd()
			Expect(ui.OutputWriter.String()).To(ContainSubstring("rocky-host-12"))
			Expect(code).To(Equal(0))
		})
	})
})
