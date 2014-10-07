package ibuild_test

import (
	"archive/tar"
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command/images/ibuild"
	"github.com/nitrous-io/nitrous-common/httpclient"
	"github.com/nitrous-io/nitrous-common/isarchive"

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
		errorWriter  *bytes.Buffer
		srv          *ghttp.Server
		handlers     http.HandlerFunc
		statusCode   int
		responseBody string
		tmpdir       string
		outfile      *os.File
		expectedSlug string
		dfBytes      = []byte("FROM scratch")
	)

	// creates and closes tmp file, returns filename, caller to remove file
	makeDockerfile := func(dfPath string) string {
		if dfPath == "" {
			dfPath = path.Join(tmpdir, "Dockerfile")
		}
		err := ioutil.WriteFile(dfPath, dfBytes, 0600)
		Expect(err).To(BeNil())
		return dfPath
	}

	// creates and closes tmp file, returns filename, caller to remove file
	makeTar := func(compress bool) string {
		f, err := ioutil.TempFile(tmpdir, "build-valid-tar")
		Expect(err).To(BeNil())
		f.Close()

		dfPath := makeDockerfile("")
		args := "cf"
		if compress {
			args = args + "z"
		}
		cmd := exec.Command("tar", args, f.Name(), path.Base(dfPath))
		cmd.Dir = tmpdir
		err = cmd.Run()
		Expect(err).To(BeNil())
		return f.Name()
	}

	// creates and returns directory, caller to remove directory
	makeDir := func() string {
		dir := path.Join(tmpdir, "scratch")
		err := os.Mkdir(dir, 0777)
		Expect(err).To(BeNil())

		// add a Dockerfile
		makeDockerfile(path.Join(dir, "Dockerfile"))

		// add a sub-directory
		subdir := path.Join(dir, "sub")
		err = os.Mkdir(subdir, 0777)
		Expect(err).To(BeNil())

		// add a file inside subdirectory
		err = ioutil.WriteFile(path.Join(subdir, "hello.txt"), []byte("hello world"), 0600)
		Expect(err).To(BeNil())

		// add a file to ignore
		err = ioutil.WriteFile(path.Join(subdir, "ignoreme"), []byte("hello"), 0600)
		Expect(err).To(BeNil())

		// add a .dockerignore file
		err = ioutil.WriteFile(path.Join(dir, ".dockerignore"), []byte("sub/ignoreme"), 0600)
		Expect(err).To(BeNil())

		return dir
	}

	BeforeEach(func() {
		var err error
		tmpdir, err = ioutil.TempDir("", "nitro-build")
		Expect(err).To(BeNil())
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
				// if a remote param is not set,
				// a file param containing a tarball is required.
				if req.FormValue("remote") == "" {
					f, _, err := req.FormFile("file")
					Expect(err).To(BeNil())
					defer f.Close()

					bufRdr := bufio.NewReader(f)
					header, _ := bufRdr.Peek(isarchive.MaxHeaderLength)
					Expect(header).NotTo(BeNil())
					Expect(isarchive.IsArchive(header)).To(BeTrue())
				}
			},
			ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
			ghttp.VerifyRequest("POST", "/api/v0/hosts/myhost/images"),
			ghttp.RespondWithPtr(&statusCode, &responseBody),
		)
		srv.AppendHandlers(handlers)

		// Create a tmp file to store build output which normally gets written
		// to stdout.
		outfile, err = ioutil.TempFile(tmpdir, "output")
		Expect(err).To(BeNil())

		errorWriter = bytes.NewBuffer(nil)
		cmd = &ibuild.Command{
			UI: &cli.BasicUi{
				Writer:      outfile,
				ErrorWriter: errorWriter,
			},
			Server: srv.URL(),
			Client: httpclient.New("correct_token"),
		}
	})

	AfterEach(func() {
		outfile.Close()
		if strings.HasPrefix(tmpdir, os.TempDir()) {
			os.RemoveAll(tmpdir)
		}
		srv.Close()
	})

	Context("nitro images:build host buildcontext", func() {
		doCmd := func(fpath string) int {
			return cmd.Run([]string{"myhost", fpath})
		}

		Context("different valid paths", func() {
			BeforeEach(func() {
				statusCode = http.StatusOK
				responseBody = `
					{"stream":"Step 0 : FROM ubuntu:latest\n"}
					{"stream":" ---\u003e 4300eb9d3c8d\n"}
					{"stream":"Step 1 : MAINTAINER docker <eng@docker.com>\n"}
					{"stream":" ---\u003e Using cache\n"}
					{"stream":" ---\u003e 3a3ed758c370\n"}
					{"stream":"Step 2 : CMD /usr/bin/top\n"}
					{"stream":" ---\u003e Running in 36b1479cc2e4\n"}
					{"stream":" ---\u003e 4b6188aebe39\n"}
					{"stream":"Removing intermediate container 36b1479cc2e4\n"}
					{"stream":"Successfully built 4b6188aebe39\n"}
			    `
			})

			It("accepts and uploads a plain Dockerfile as a tar", func() {
				fpath := makeDockerfile("")
				code := doCmd(fpath)
				Expect(code).To(Equal(0))
				Expect(srv.ReceivedRequests()).To(HaveLen(1))
				output, _ := ioutil.ReadFile(outfile.Name())
				Expect(string(output)).To(ContainSubstring("Step 0 : FROM ubuntu:latest"))
				Expect(string(output)).To(ContainSubstring("Step 1 : MAINTAINER docker <eng@docker.com>"))
				Expect(string(output)).To(ContainSubstring("Successfully built 4b6188aebe39"))

				Expect(string(output)).To(ContainSubstring("Built image"))
			})

			It("accepts and uploads a compressed tarball", func() {
				fpath := makeTar(true)
				code := doCmd(fpath)
				Expect(srv.ReceivedRequests()).To(HaveLen(1))
				output, _ := ioutil.ReadFile(outfile.Name())
				Expect(code).To(Equal(0))
				Expect(string(output)).To(ContainSubstring("Built image"))
			})

			It("accepts and uploads uncompressed tarball", func() {
				fpath := makeTar(false)
				code := doCmd(fpath)
				Expect(srv.ReceivedRequests()).To(HaveLen(1))
				output, _ := ioutil.ReadFile(outfile.Name())
				Expect(code).To(Equal(0))
				Expect(string(output)).To(ContainSubstring("Built image"))
			})

			It("accepts a directory with a Dockerfile, creates and uploads it as a tar", func() {
				fpath := makeDir()
				code := doCmd(fpath)
				reqs := srv.ReceivedRequests()
				Expect(reqs).To(HaveLen(1))

				f, _, err := reqs[0].FormFile("file")
				Expect(err).To(BeNil())
				defer f.Close()

				tr := tar.NewReader(f)

				expectedPaths := []string{
					"./",
					"Dockerfile",
					".dockerignore",
					"sub/",
					"sub/hello.txt",
				}
				for {
					hdr, err := tr.Next()
					if err == io.EOF {
						break
					}
					Expect(err).To(BeNil())
					Expect(expectedPaths).To(ContainElement(hdr.Name))
				}

				output, _ := ioutil.ReadFile(outfile.Name())
				Expect(code).To(Equal(0))
				Expect(string(output)).To(ContainSubstring("Built image"))
			})

			Context("when given path is a valid URL", func() {
				for _, t := range []string{
					"https://github.com/nitrous-io/docker-hello.git",
					"git://github.com/nitrous-io/docker-hello",
					"git@github.com:nitrous-io/docker-hello",
					"http://pastebin.com/raw.php?i=some-dockerfile",
					"github.com/nitrous-io/docker-hello",
					"user@bitbucket.org/nitrous-io/docker-hello.git",
				} {
					url := t
					It("makes the request with URL set in remote param "+url, func() {
						code := doCmd(url)
						reqs := srv.ReceivedRequests()
						Expect(reqs).To(HaveLen(1))
						Expect(reqs[0].FormValue("remote")).To(Equal(url))

						output, _ := ioutil.ReadFile(outfile.Name())
						Expect(code).To(Equal(0))
						Expect(string(output)).To(ContainSubstring("Built image"))
					})
				}
			})
		})

		Context("with slug specified", func() {
			BeforeEach(func() {
				statusCode = http.StatusOK
				responseBody = `
					{"stream":"Step 0 : FROM ubuntu:latest\n"}
					{"stream":" ---\u003e 4300eb9d3c8d\n"}
					{"stream":"Step 1 : MAINTAINER docker <eng@docker.com>\n"}
					{"stream":" ---\u003e Using cache\n"}
					{"stream":" ---\u003e 3a3ed758c370\n"}
					{"stream":"Step 2 : CMD /usr/bin/top\n"}
					{"stream":" ---\u003e Running in 36b1479cc2e4\n"}
					{"stream":" ---\u003e 4b6188aebe39\n"}
					{"stream":"Removing intermediate container 36b1479cc2e4\n"}
					{"stream":"Successfully built 4b6188aebe39\n"}
			    `
				expectedSlug = "UseThisSlug"
			})

			It("uses slug", func() {
				fpath := makeDockerfile("")
				code := cmd.Run([]string{"myhost", fpath, "--slug=UseThisSlug"})
				Expect(code).To(Equal(0))
				Expect(srv.ReceivedRequests()).To(HaveLen(1))
				output, _ := ioutil.ReadFile(outfile.Name())
				Expect(string(output)).To(ContainSubstring("Built image"))
			})
		})

		Context("error on build context not existing", func() {
			It("outputs error message", func() {
				code := doCmd("/tmp/nosuch")
				Expect(code).To(Equal(1))
				Expect(srv.ReceivedRequests()).To(HaveLen(0))
				Expect(errorWriter.String()).To(ContainSubstring("path is invalid"))
			})
		})

		Context("JSON response with error", func() {
			BeforeEach(func() {
				statusCode = http.StatusInternalServerError
				responseBody = `
					{"stream":"Step 0 : FROM broken\n"}
					{"stream":" ---\u003e 4300eb9d3c8d\n"}
					{"error":"Error...", "errorDetail":{"code": 123, "message": "Error pulling 'broken'"}}
			    `
			})

			It("outputs error message", func() {
				fpath := makeDockerfile("")
				code := doCmd(fpath)
				Expect(code).To(Equal(1))
				Expect(srv.ReceivedRequests()).To(HaveLen(1))
				output, _ := ioutil.ReadFile(outfile.Name())
				Expect(string(output)).NotTo(ContainSubstring("Step 0 : FROM broken")) // Stream is not displayed
				Expect(errorWriter.String()).NotTo(BeEmpty())
			})
		})
	})

	Context("when given wrong arguments", func() {
		It("prints usage and exits with error", func() {
			for _, t := range [][]string{{}, {"hostonly"}, {"host", "buildcontext", "somerandomextrastuff"}} {
				code := cmd.Run(t)
				Expect(code).To(Equal(1))
				output, _ := ioutil.ReadFile(outfile.Name())
				Expect(string(output)).To(ContainSubstring("Usage:"))
			}
		})
	})

	Context("Help", func() {
		It("outputs usage", func() {
			Expect(cmd.Help()).To(ContainSubstring("Usage: nitro images:build"))
		})
	})
})
