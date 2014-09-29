package create_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command/containers/create"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "containers/create")
}

var _ = Describe("containers/create", func() {
	var (
		cmd      *create.Command
		outfile  *os.File
		errorBuf *bytes.Buffer

		slug = "foo"

		srv                *ghttp.Server
		pullStatusCode     int
		pullResponseBody   string
		createStatusCode   int
		createResponseBody string
	)

	BeforeEach(func() {
		var err error

		// Setup for success.
		pullStatusCode = http.StatusOK
		createStatusCode = http.StatusCreated

		srv = ghttp.NewServer()

		// Setup handlers: pull image.
		srv.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyHeaderKV("Authorization", "Bearer a_token"),
			ghttp.VerifyRequest("POST", "/api/v0/hosts/host-1/images/pull"),
			func(w http.ResponseWriter, req *http.Request) {
				req.ParseForm()
				Expect(req.PostFormValue("update")).To(Equal("false"))
			},
			ghttp.RespondWithPtr(&pullStatusCode, &pullResponseBody),
		))

		// Setup handlers: create container.
		srv.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyHeaderKV("Authorization", "Bearer a_token"),
			ghttp.VerifyRequest("POST", "/api/v0/containers"),
			func(w http.ResponseWriter, req *http.Request) {
				req.ParseForm()
				Expect(req.PostFormValue("host")).To(Equal("host-1"))
			},
			ghttp.RespondWithPtr(&createStatusCode, &createResponseBody),
		))

		// Create a tmp file to store output.
		outfile, err = ioutil.TempFile("/tmp", "images-pull")
		Expect(err).To(BeNil(), "failed to create tmp file for storing output")

		errorBuf = bytes.NewBuffer(nil)
		cmd = &create.Command{
			UI: &cli.BasicUi{
				Writer:      outfile,
				ErrorWriter: errorBuf,
			},
			Server: srv.URL(),
			Client: httpclient.New("a_token"),
		}
	})

	AfterEach(func() {
		srv.Close()
		os.RemoveAll(outfile.Name())
	})

	doCmd := func() int {
		return cmd.Run([]string{
			"host-1",
			"--slug=" + slug,
			"--image=ubuntu",
			"--command=/bin/dash -c \"echo 'hi'\"",
			"--env=PAGER=most",
			"--volume=/shared", // repeat args supported
			"--volume=/music",
			"--port-forward=8080:3000,127.0.0.1:4242", // comma-separated supported
			"--port-forward-all",
		})
	}

	Context("when not all options are specified", func() {
		It("populates request with only given params", func() {
			cmd.Run([]string{
				"host-1",
				"--slug=" + slug,
			})
			reqs := srv.ReceivedRequests()
			Expect(reqs).To(HaveLen(2))
			Expect(reqs[1].PostFormValue("slug")).To(Equal(slug))
		})
	})

	Context("when slug is not specified", func() {
		It("populates request with a randomly generated slug", func() {
			cmd.Run([]string{
				"host-1",
			})
			reqs := srv.ReceivedRequests()
			Expect(reqs).To(HaveLen(2))
			Expect(reqs[1].PostFormValue("slug")).NotTo(Equal(""))
		})
	})

	Context("when host is not specified", func() {
		It("returns an error message without making a request", func() {
			code := cmd.Run([]string{
				"--slug=" + slug,
				"--image=ubuntu",
			})
			Expect(errorBuf.String()).To(ContainSubstring("Please specify a host"))
			Expect(code).To(Equal(1))
			Expect(srv.ReceivedRequests()).To(HaveLen(0))
		})
	})

	Context("when cmd is not specified", func() {
		It("should be omitted from the request", func() {
			cmd.Run([]string{
				"host-1",
				"--slug=" + slug,
			})
			reqs := srv.ReceivedRequests()
			Expect(reqs).To(HaveLen(2))
			_, found := reqs[1].Form["cmd"]
			Expect(found).To(BeFalse())
		})
	})

	Context("when pull image request fails", func() {
		It("does not make the subsequent create request", func() {
			pullStatusCode = http.StatusInternalServerError

			code := doCmd()
			Expect(code).To(Equal(1))
			reqs := srv.ReceivedRequests()
			Expect(reqs).To(HaveLen(1))
			r := reqs[0]
			Expect(r.PostFormValue("repository")).To(Equal("ubuntu"))
		})
	})

	Context("when pull image request succeeds", func() {
		It("makes a create request", func() {
			pullStatusCode = http.StatusOK

			code := doCmd()
			Expect(code).To(Equal(1))
			reqs := srv.ReceivedRequests()
			Expect(reqs).To(HaveLen(2))
			pullReq, createReq := reqs[0], reqs[1]

			Expect(pullReq.PostFormValue("repository")).To(Equal("ubuntu"))

			Expect(createReq.PostFormValue("slug")).To(Equal(slug))
			Expect(createReq.PostFormValue("image")).To(Equal("ubuntu"))
			Expect(createReq.PostFormValue("cmd")).To(Equal("/bin/dash -c \"echo 'hi'\""))
			Expect(createReq.PostForm["env[]"]).To(Equal([]string{"PAGER=most"}))
			Expect(createReq.PostForm["volume[]"]).To(Equal([]string{"/shared", "/music"}))
			Expect(createReq.PostForm["port[]"]).To(Equal([]string{"8080:3000", "127.0.0.1:4242"}))
		})
	})

	Context("when pull image request succeeds but create request fails", func() {
		It("does not make the subsequent start request", func() {
			createStatusCode = http.StatusBadRequest
			createResponseBody = `{"error":"validation_failed","error_description":"slug invalid"}`

			code := doCmd()
			Expect(errorBuf.String()).To(ContainSubstring("slug invalid"))
			Expect(code).To(Equal(1))

			reqs := srv.ReceivedRequests()
			Expect(reqs).To(HaveLen(2))
		})
	})

	Context("when pull image and create requests succeed", func() {
		It("prints a message indicating that the container was created", func() {
			pullStatusCode = http.StatusOK
			createStatusCode = http.StatusCreated
			createResponseBody = `{
				"container": { "slug": "` + slug + `", "image": "ubuntu", "state": "running" }
			}`

			code := doCmd()
			Expect(code).To(Equal(0))

			output, _ := ioutil.ReadFile(outfile.Name())
			Expect(output).To(ContainSubstring("Created new container '" + slug + "'"))
		})
	})

	Context("when pull image and create requests succeed, but create request returns an error description", func() {
		It("prints a message indicating that the container was created but with errors", func() {
			createResponseBody = `{
				"error_description": "start failed: could not bind port 22",
				"container": { "slug": "` + slug + `", "image": "ubuntu", "state": "running" }
			}`

			code := doCmd()
			Expect(code).To(Equal(0))

			output, _ := ioutil.ReadFile(outfile.Name())
			Expect(output).To(ContainSubstring("Created new container '" + slug + "' with errors: start failed: could not bind port 22"))
		})
	})
})
