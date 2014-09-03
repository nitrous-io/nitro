package ibuild_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/images/ibuild"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "images/ibuild")
}

var _ = Describe("images/ibuild", func() {
	var (
		cmd          *ibuild.Command
		ui           *command.MockUI
		srv          *ghttp.Server
		handlers     http.HandlerFunc
		statusCode   int
		responseBody string
		tmpdir       string
		expectedSlug string
	)

	makeFile := func() string {
		tmpdir, err := ioutil.TempDir("", "nitro-build")
		Expect(err).To(BeNil())

		path := filepath.Join(tmpdir, "Dockerfile")
		err = ioutil.WriteFile(path, []byte("FROM scratch"), 0600)
		Expect(err).To(BeNil())
		return path
	}

	BeforeEach(func() {
		var err error
		tmpdir, err = ioutil.TempDir("", "nitro-build")
		Expect(err).To(BeNil())
		ui = &command.MockUI{&cli.MockUi{}}
		srv = ghttp.NewServer()
		expectedSlug = ""
		handlers = ghttp.CombineHandlers(
			func(w http.ResponseWriter, req *http.Request) {
				if expectedSlug != "" {
					Expect(req.FormValue("slug")).To(Equal(expectedSlug))
				} else {
					Expect(req.FormValue("slug")).NotTo(BeEmpty())
				}
			},
			func(w http.ResponseWriter, req *http.Request) {
				// check if the request has 'file' param set
				f, _, err := req.FormFile("file")
				Expect(err).To(BeNil())
				defer f.Close()

				dfBytes := []byte("FROM scratch")
				buf := make([]byte, len(dfBytes))
				_, err = f.Read(buf)
				Expect(err).To(BeNil())
				Expect(buf).To(Equal(dfBytes))
			},
			ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
			ghttp.VerifyRequest("POST", "/api/v0/hosts/myhost/images"),
			ghttp.RespondWithPtr(&statusCode, &responseBody),
		)
		srv.AppendHandlers(handlers)
		cmd = &ibuild.Command{
			UI:     ui,
			Server: srv.URL(),
			Client: httpclient.New("correct_token"),
		}
	})

	AfterEach(func() {
		if strings.HasPrefix(tmpdir, os.TempDir()) {
			os.RemoveAll(tmpdir)
		}
		srv.Close()
	})

	Context("nitro images:build host buildcontext", func() {
		doCmd := func(fpath string) int {
			return cmd.Run([]string{"myhost", fpath})
		}

		Context("successful build", func() {
			BeforeEach(func() {
				statusCode = http.StatusOK
				responseBody = `{"status":"OK"}`
			})

			It("uploads the given file", func() {
				fpath := makeFile()
				code := doCmd(fpath)
				Expect(ui.OutputWriter.String()).To(ContainSubstring("Built image"))
				Expect(code).To(Equal(0))
				Expect(srv.ReceivedRequests()).To(HaveLen(1))
			})
		})

		Context("with slug specified", func() {
			BeforeEach(func() {
				statusCode = http.StatusOK
				responseBody = `{"status":"OK"}`
				expectedSlug = "UseThisSlug"
			})

			It("uses slug", func() {
				fpath := makeFile()
				code := cmd.Run([]string{"myhost", fpath, "--slug=UseThisSlug"})
				Expect(ui.OutputWriter.String()).To(ContainSubstring("Built image"))
				Expect(code).To(Equal(0))
				Expect(srv.ReceivedRequests()).To(HaveLen(1))
			})
		})

		Context("error on build context not existing", func() {
			It("outputs error message", func() {
				code := doCmd("/tmp/nosuch")
				Expect(ui.ErrorWriter.String()).To(ContainSubstring("no such file"))
				Expect(code).To(Equal(1))
				Expect(srv.ReceivedRequests()).To(HaveLen(0))
			})
		})

		Context("JSON response with error", func() {
			BeforeEach(func() {
				statusCode = http.StatusBadRequest
				responseBody = `{"error": "validation_failed", "error_description": "image not found"}`
			})

			It("outputs error message", func() {
				fpath := makeFile()
				code := doCmd(fpath)
				Expect(ui.ErrorWriter.String()).To(Equal("image not found\n"))
				Expect(code).To(Equal(1))
				Expect(srv.ReceivedRequests()).To(HaveLen(1))
			})
		})
	})

	Context("when given wrong arguments", func() {
		It("prints usage and exits with error", func() {
			for _, t := range [][]string{{}, {"hostonly"}, {"host", "buildcontext", "somerandomextrastuff"}} {
				code := cmd.Run(t)
				Expect(ui.OutputWriter.String()).To(ContainSubstring("Usage:"))
				Expect(code).To(Equal(1))
			}
		})
	})

	Context("Help", func() {
		It("outputs usage", func() {
			Expect(cmd.Help()).To(ContainSubstring("Usage: nitro images:build"))
		})
	})
})
