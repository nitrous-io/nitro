package inspect_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/containers/inspect"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "containers/inspect")
}

var _ = Describe("containers/inspect", func() {
	var (
		cmd *inspect.Command
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
			ghttp.VerifyRequest("GET", "/api/v0/containers/wild-rocket-23"),
			ghttp.RespondWithPtr(&statusCode, &responseBody),
		)
		srv = ghttp.NewServer()
		srv.AppendHandlers(handlers)

		cmd = &inspect.Command{
			UI:     ui,
			Server: srv.URL(),
			Client: httpclient.New("correct_token"),
		}

		command.Env.AuthToken = "a_token"
	})

	AfterEach(func() {
		srv.Close()
	})

	Context("nitro inspect containerslug", func() {
		doCmd := func() int {
			return cmd.Run([]string{"wild-rocket-23"})
		}

		Context("when the API responds with the container", func() {
			BeforeEach(func() {
				statusCode = http.StatusOK
				responseBody = `{ "container": { "slug": "wild-rocket-23", "image": "ubuntu", "state": "running" } }`
			})

			It("prints the container", func() {
				code := doCmd()
				Expect(ui.OutputWriter.String()).To(ContainSubstring(strings.TrimSpace(`
Slug              Image     State
wild-rocket-23    ubuntu    running
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
			code := cmd.Run([]string{"host-1/wild-rocket-23", "hey", "man"})
			Expect(ui.OutputWriter.String()).To(ContainSubstring("Usage:"))
			Expect(code).To(Equal(1))
		})
	})
})
