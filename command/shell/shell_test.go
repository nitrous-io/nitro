package shell

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/kr/pty"
	"github.com/mitchellh/cli"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func hijackServer(w http.ResponseWriter) (io.ReadCloser, io.Writer, error) {
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		return nil, nil, err
	}
	// Flush the options to make sure the client sets the raw mode
	fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Type: application/vnd.containerd.raw-stream\r\n\r\n")
	return conn, conn, nil
}

type testServer struct {
	srv *httptest.Server

	containerId string
	shellId     string

	stdin, stdout *bytes.Buffer
	resized       bool
	finished      bool
}

func (s *testServer) createShell(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	Expect(id).To(Equal(s.containerId))
	json.NewEncoder(w).Encode(map[string]string{
		"id": s.shellId,
	})
}

func (s *testServer) attachShell(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	Expect(id).To(Equal(s.shellId))
	stdin, stdout, _ := hijackServer(w)
	go io.Copy(s.stdin, stdin)
	io.Copy(stdout, s.stdout)
	s.finished = true
}

func (s *testServer) resizeShell(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	Expect(id).To(Equal(s.shellId))
	s.resized = true
	w.WriteHeader(http.StatusAccepted)
}

func (s *testServer) Close() {
	s.srv.CloseClientConnections()
	s.srv.Close()
}

func newTestServer(containerId, shellId string) *testServer {
	s := &testServer{
		containerId: containerId,
		shellId:     shellId,
		stdin:       bytes.NewBuffer(nil),
		stdout:      bytes.NewBuffer(nil),
	}
	r := mux.NewRouter()
	r.HandleFunc("/containers/{id}/shells", s.createShell).Methods("POST")
	r.HandleFunc("/shells/{id}/attach", s.attachShell).Methods("POST")
	r.HandleFunc("/shells/{id}/resize", s.resizeShell).Methods("POST")

	srv := httptest.NewServer(r)
	s.srv = srv
	return s
}

func TestShell(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "shell")
}

var _ = Describe("ShellCommand", func() {
	It("requires host and container name", func() {
		testCases := [][]string{
			{},
			{"only-host"},
		}
		for _, tc := range testCases {
			cmd := &ShellCommand{UI: &cli.MockUi{}}
			code := cmd.Run(tc)
			Expect(code).To(Equal(1), "Expect command to fail with status %d", 1)
		}
	})

	It("attaches to the remote shell", func() {
		containerID := "best-container"
		ts := newTestServer(containerID, "shell-id")
		defer ts.Close()

		stdin, stdout, _ := pty.Open()
		ui := &cli.BasicUi{
			Reader:      stdin,
			Writer:      stdout,
			ErrorWriter: bytes.NewBuffer(nil),
		}
		cmd := &ShellCommand{UI: ui}

		go func() {
			defer GinkgoRecover()
			cmd.Run([]string{ts.srv.Listener.Addr().String(), containerID})
			Expect(ui.ErrorWriter.(*bytes.Buffer).String()).To(BeEmpty())
		}()

		Eventually(func() bool {
			return ts.resized
		}).Should(BeTrue())

		Eventually(func() bool {
			return ts.finished
		}).Should(BeTrue())
	})

	Describe("getLoginCommand", func() {
		It("returns the logins command", func() {
			testCases := [][]string{
				{"", "sudo login -f root"},
				{"action", "sudo login -f action"},
			}

			for _, tc := range testCases {
				cmd := getLoginCommand(tc[0])
				Expect(cmd).To(Equal(tc[1]), "Incorrect login command")
			}
		})
	})
})
