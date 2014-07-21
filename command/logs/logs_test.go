package logs

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/mitchellh/cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLogs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "logs")
}

var _ = Describe("logs", func() {
	It("requires host and container id", func() {
		testCases := [][]string{
			{},
			{"only-host"},
		}
		for _, tc := range testCases {
			cmd := &Command{
				UI: &cli.BasicUi{ErrorWriter: bytes.NewBuffer(nil)},
			}
			code := cmd.Run(tc)
			Expect(code).To(Equal(1), "Requires hosts and containers but got %v", tc)
		}
	})

	It("handles default value for follow and timestamps", func() {
		ts := newTestServer("container")
		defer ts.Close()

		testCases := []struct {
			args           []string
			expectedParams string
		}{
			{[]string{ts.srv.URL, "container"}, ""},
			{[]string{"-f", ts.srv.URL, "container"}, "follow=1"},
			{[]string{"-t", ts.srv.URL, "container"}, "timestamps=1"},
			{[]string{"-f", "-t", ts.srv.URL, "container"}, "follow=1&timestamps=1"},
		}

		for _, tc := range testCases {
			cmd := &Command{
				UI: &cli.BasicUi{
					Writer:      bytes.NewBuffer(nil),
					ErrorWriter: bytes.NewBuffer(nil),
				},
			}
			cmd.Run(tc.args)
			Expect(ts.lastForm.Encode()).To(Equal(tc.expectedParams))
			GinkgoRecover()
		}
	})
})

type testServer struct {
	srv         *httptest.Server
	containerId string
	output      *bytes.Buffer
	lastForm    url.Values
}

func (s *testServer) getLogs(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	Expect(id).To(Equal(s.containerId))
	r.ParseForm()
	s.lastForm = r.Form
	io.Copy(w, s.output)
}

func (s *testServer) Close() {
	s.srv.CloseClientConnections()
	s.srv.Close()
}

func newTestServer(containerId string) *testServer {
	s := &testServer{
		containerId: containerId,
		output:      bytes.NewBuffer(nil),
	}
	r := mux.NewRouter()
	r.HandleFunc("/containers/{id}/logs", s.getLogs).Methods("GET")

	srv := httptest.NewServer(r)
	s.srv = srv
	return s
}
