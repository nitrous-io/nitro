package export

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mitchellh/cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestExport(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "export")
}

var _ = Describe("Export", func() {
	Describe("getLocalArchive", func() {
		It("returns a local file suitable for writing", func() {
			f, err := getLocalArchive("test-host", "")
			defer os.RemoveAll(f.Name())
			Expect(err).To(BeNil())
			_, err = f.Stat()
			Expect(err).To(BeNil(), "Expected the local archive to be created, got err %v", err)
		})
	})

	Describe("downloadArchive", func() {
		It("fails when the container id is invalid", func() {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(nil))
			}))
			defer srv.Close()

			w := bytes.NewBuffer(nil)
			err := downloadArchive(&cli.MockUi{}, w, srv.URL, "invalid-id")
			Expect(err).ToNot(BeNil(), "Expect errors when container is not valid")
		})

		It("downloads when the container is valid", func() {
			content := "archive-content"
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(content))
			}))
			defer srv.Close()

			w := bytes.NewBuffer(nil)
			err := downloadArchive(&cli.MockUi{}, w, srv.URL, "container-id")
			Expect(err).To(BeNil(), "Failed to download archive. Unexpected error %v", err)
			Expect(w.String()).To(Equal(content))
		})
	})
})
