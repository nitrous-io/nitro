package list_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/hosts/list"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "hosts/list")
}

var _ = Describe("hosts/list", func() {
	var (
		cmd      *list.Command
		ui       *cli.MockUi
		srv      *ghttp.Server
		handlers []http.HandlerFunc
	)

	BeforeEach(func() {
		command.Env.AuthToken = "a_token"

		srv = ghttp.NewServer()
		ui = &cli.MockUi{}
		cmd = &list.Command{
			UI:     ui,
			Server: srv.URL(),
			Client: httpclient.New("a_token"),
		}
		handlers = []http.HandlerFunc{
			ghttp.VerifyHeader(http.Header{
				"Authorization": []string{"Bearer a_token"},
			}),
			ghttp.VerifyRequest("GET", "/api/v0/hosts"),
		}

		command.Env.AuthToken = "a_token"
	})

	AfterEach(func() {
		srv.Close()
	})

	Context("hitting the server", func() {
		Context("when server responds with hosts", func() {
			var (
				responseJSON = strings.TrimSpace(`
				{"hosts":[{"ID":1,"UserID":1,"RegionID":1,"PlanID":1,"ProviderTemplateID":1,"slug":"host-1","name":{"String":"","Valid":false},"State":"born","IPAddress":{"String":"","Valid":false},"ProviderIdentifier":{"String":"","Valid":false},"MaxContainers":{"Int64":0,"Valid":false},"FreighterdPort":{"Int64":0,"Valid":false},"TerminatedAt":null,"CreatedAt":"2014-07-10T12:05:14.473679Z","UpdatedAt":"2014-07-10T12:05:14.473679Z"},{"ID":2,"UserID":1,"RegionID":2,"PlanID":1,"ProviderTemplateID":2,"slug":"host-2","name":{"String":"","Valid":false},"State":"running","IPAddress":{"String":"","Valid":false},"ProviderIdentifier":{"String":"","Valid":false},"MaxContainers":{"Int64":0,"Valid":false},"FreighterdPort":{"Int64":0,"Valid":false},"TerminatedAt":null,"CreatedAt":"2014-07-10T12:05:14.473679Z","UpdatedAt":"2014-07-10T12:05:14.473679Z"}]}
	`)

				formattedOutput = strings.TrimSpace(`
Slug      State
host-1    born
host-2    running
`) + "\n\n"
			)

			BeforeEach(func() {
				handlers = append(handlers,
					ghttp.RespondWith(http.StatusOK, responseJSON))
				srv.AppendHandlers(ghttp.CombineHandlers(handlers...))
			})

			It("outputs formatted hosts on success", func() {
				code := cmd.Run([]string{})
				Expect(ui.OutputWriter.String()).To(Equal(formattedOutput))
				Expect(code).To(Equal(0))
			})
		})

		Context("when user has no hosts", func() {
			BeforeEach(func() {
				handlers = append(handlers,
					ghttp.RespondWith(http.StatusOK, `{"hosts":[],"status":"ok"}`))
				srv.AppendHandlers(ghttp.CombineHandlers(handlers...))
			})

			It("displays no hosts found", func() {
				code := cmd.Run([]string{})
				Expect(ui.OutputWriter.String()).To(Equal("No hosts found\n"))
				Expect(code).To(Equal(0))
			})
		})

		Context("when server responds with an internal_server_error", func() {
			BeforeEach(func() {
				handlers = append(handlers,
					ghttp.RespondWith(http.StatusInternalServerError,
						`{"error":"internal_server_error","error_hash":"b482fd5b602650c798366fe9ee8e6cf65ef759fc"}`),
				)
				srv.AppendHandlers(ghttp.CombineHandlers(handlers...))
			})

			It("displays unknown error message", func() {
				code := cmd.Run([]string{})
				Expect(ui.ErrorWriter.String()).To(ContainSubstring("Unknown error occurred"))
				Expect(code).To(Equal(1))
			})
		})
	})
})
