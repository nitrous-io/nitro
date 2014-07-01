package export

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mitchellh/cli"
)

func TestGetLocalArchive(t *testing.T) {
	f, err := getLocalArchive("test-host", "")
	defer os.RemoveAll(f.Name())
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	if _, err := f.Stat(); err != nil {
		t.Fatalf("Expected the local archive to be created, got err %v", err)
	}
}

func TestDownloadArchive(t *testing.T) {
	containerID := "test-host"
	content := "archived-content"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/containers/test-host/export" {
			w.Write([]byte(content))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(nil))
		}
	}))
	defer srv.Close()

	testCases := []struct {
		id       string
		hasError bool
	}{
		{"invalid-id", true},
		{containerID, false},
	}

	for _, tc := range testCases {
		w := bytes.NewBuffer(nil)
		err := downloadArchive(&cli.MockUi{}, w, srv.URL, tc.id)
		if tc.hasError {
			if err == nil {
				t.Errorf("Expect error when export container %s", tc.id)
			}
		} else {
			if err != nil {
				t.Errorf("Failed to download archive. Unexpected error %v", err)
			}
			if downloaded := w.String(); downloaded != content {
				t.Errorf("Expect to output %s, but got %s", content, downloaded)
			}
		}
	}
}
