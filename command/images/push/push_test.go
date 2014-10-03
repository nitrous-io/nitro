package push_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
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
		outfileName  string
		ew           *bytes.Buffer
		srv          *ghttp.Server
		statusCode   int
		responseBody string
	)

	BeforeEach(func() {
		srv = ghttp.NewServer()
		srv.AppendHandlers(ghttp.CombineHandlers(
			func(w http.ResponseWriter, req *http.Request) {
				Expect(req.FormValue("repository")).To(Equal("username/myscratch"))
			},
			ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
			ghttp.VerifyRequest("POST", "/api/v0/hosts/myhost/images/scratch/push"),
			ghttp.RespondWithPtr(&statusCode, &responseBody),
		))

		// Create a tmp file to store push output.
		out, err := ioutil.TempFile("/tmp", "images-pull")
		Expect(err).To(BeNil(), "failed to create tmp file for storing output")
		outfileName = out.Name()

		ew = bytes.NewBuffer(nil)

		cmd = &push.Command{
			UI: &cli.BasicUi{
				Writer:      out,
				ErrorWriter: ew,
			},
			Server: srv.URL(),
			Client: httpclient.New("correct_token"),
		}
	})

	AfterEach(func() {
		os.RemoveAll(outfileName)
		srv.Close()
	})

	Context("nitro images:push host image", func() {
		doCmd := func() int {
			return cmd.Run([]string{"myhost", "scratch", "username/myscratch"})
		}

		BeforeEach(func() {
			statusCode = http.StatusOK
			responseBody = `
			{"status":"The push refers to a repository [hub.docker.com/me/imageid] (len: 1)"}
			{"status":"Sending image list"}
			{"status":"Pushing repository hub.docker.com/me/imageid (1 tags)"}
			{"status":"Image 511136ea3c5a already pushed, skipping"}
			{"status":"Pushing","progressDetail":{},"id":"2e9d5ac58122"}{"status":"Buffering to disk","progressDetail":{"current":8192,"start":1412070481},"progress":"8.192 kB","id":"2e9d5ac58122"}{"status":"Buffering to disk","progressDetail":{"start":1412070481},"id":"2e9d5ac58122"}{"status":"Pushing","progressDetail":{"current":512,"total":8192,"start":1412070481},"progress":"[===\u003e                                               ]    512 B/8.192 kB 29s","id":"2e9d5ac58122"}
			`
		})

		It("prints the push progress and a success message", func() {
			code := doCmd()
			Expect(code).To(Equal(0))

			output, _ := ioutil.ReadFile(outfileName)

			// Assert that output contains parsed output and not the raw JSON
			// stream.
			Expect(string(output)).To(ContainSubstring(strings.TrimSpace(`
The push refers to a repository [hub.docker.com/me/imageid] (len: 1)
Sending image list
Pushing repository hub.docker.com/me/imageid (1 tags)
Image 511136ea3c5a already pushed, skipping`)))
			Expect(string(output)).To(ContainSubstring(`2e9d5ac58122: Pushing`))
			Expect(string(output)).To(ContainSubstring(`512 B/8.192 kB`))

			Expect(string(output)).To(ContainSubstring("Pushed image"))
		})

		Context("when request fails with a HTTP 401 Unauthorized", func() {
			BeforeEach(func() {
				statusCode = http.StatusUnauthorized
				responseBody = `{"error": "invalid_request", "error_description": "don't care"}`
			})

			It("prints failure and a hint to login", func() {
				code := doCmd()
				Expect(ew.String()).To(ContainSubstring("Failed: "))
				Expect(ew.String()).To(ContainSubstring("nitro registry:login"))
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
				Expect(ew.String()).To(Equal("image not found\n"))
				Expect(code).To(Equal(1))
			})
		})
	})

	Context("when given wrong arguments", func() {
		It("prints usage and exits with error", func() {
			for _, t := range [][]string{{}, {"hostonly"}, {"host", "image"}} {
				code := cmd.Run(t)
				output, _ := ioutil.ReadFile(outfileName)
				Expect(string(output)).To(ContainSubstring("Usage:"))
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
