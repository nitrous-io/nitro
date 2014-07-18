package create_test

import (
	"net/http"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/hosts/create"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "hosts/create")
}

var _ = Describe("hosts/create", func() {
	var (
		cmd      *create.Command
		ui       *cli.MockUi
		srv      *ghttp.Server
		handlers []http.HandlerFunc
	)

	BeforeEach(func() {
		srv = ghttp.NewServer()
		ui = &cli.MockUi{}
		cmd = &create.Command{
			UI:     ui,
			Server: srv.URL(),
			Client: httpclient.New("a_token"),
		}
		handlers = []http.HandlerFunc{
			ghttp.VerifyHeader(http.Header{
				"Authorization": []string{"Bearer a_token"},
			}),
			ghttp.VerifyRequest("POST", "/api/v0/hosts"),
		}

		command.Env.AuthToken = "a_token"
	})

	It("requires size as an argument", func() {
		code := cmd.Run([]string{})
		Expect(code).To(Equal(1))
		Expect(ui.OutputWriter.String()).To(ContainSubstring("Usage:"))
	})

	Context("hitting the server", func() {
		type createHostParams struct {
			size   string
			region string
			slug   string
		}

		doCmd := func(params *createHostParams) int {
			return cmd.Run([]string{params.size,
				"-r", params.region,
				"-s", params.slug,
			})
		}

		Context("when slug is provided", func() {
			It("populates request correctly", func() {
				handlers = append(handlers, func(w http.ResponseWriter, r *http.Request) {
					Expect(r.FormValue("size")).To(Equal("micro"))
					Expect(r.FormValue("region")).To(Equal("apse"))
					Expect(r.FormValue("slug")).To(Equal("foo"))
				})
				srv.AppendHandlers(ghttp.CombineHandlers(handlers...))
				doCmd(&createHostParams{size: "micro", slug: "foo", region: "apse"})
			})
		})

		Context("when slug is not provided", func() {
			It("populates request with a randomly generated slug", func() {
				handlers = append(handlers, func(w http.ResponseWriter, r *http.Request) {
					Expect(r.FormValue("size")).To(Equal("micro"))
					Expect(r.FormValue("region")).To(Equal("apse"))
					// This will be a randomly generated name
					Expect(r.FormValue("slug")).NotTo(Equal(""))

				})
				srv.AppendHandlers(ghttp.CombineHandlers(handlers...))
				doCmd(&createHostParams{size: "micro", region: "apse"})
			})

		})

		Context("when request succeeds", func() {
			It("reports success", func() {
				handlers = append(handlers,
					ghttp.RespondWith(http.StatusCreated, `{"status":"created"}`))
				srv.AppendHandlers(ghttp.CombineHandlers(handlers...))

				code := doCmd(&createHostParams{size: "micro", slug: "test", region: "apse"})
				Expect(ui.OutputWriter.String()).To(ContainSubstring("Done"))
				Expect(code).To(Equal(0))
			})
		})

		Context("when request fails", func() {
			It("displays error message", func() {
				handlers = append(handlers,
					ghttp.RespondWith(http.StatusBadRequest,
						`{"error":"validation_failed","error_description":"region not found"}`),
				)
				srv.AppendHandlers(ghttp.CombineHandlers(handlers...))

				code := doCmd(&createHostParams{size: "micro", slug: "test", region: "apse"})
				Expect(ui.ErrorWriter.String()).To(ContainSubstring("region not found"))
				Expect(code).To(Equal(1))
			})
		})
	})
})
