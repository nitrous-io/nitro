package shelltest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/dotcloud/docker/vendor/src/github.com/kr/pty"
	"github.com/gorilla/mux"
	"github.com/mitchellh/cli"
	. "github.com/nitrous-io/nitro/command/shell"

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

var _ = Describe("ShellCommand", func() {
	var (
		cmd *ShellCommand
		ui  *cli.BasicUi

		host        string
		containerId string
		args        []string
		shellId     string

		srv *testServer

		stdin, stdout *os.File
	)

	BeforeEach(func() {
		stdin, stdout, _ = pty.Open()
		ui = &cli.BasicUi{
			Reader:      stdin,
			Writer:      stdout,
			ErrorWriter: bytes.NewBuffer(nil),
		}
		cmd = &ShellCommand{UI: ui}

		containerId = "best-container"
		shellId = "current-shell"

		srv = newTestServer(containerId, shellId)

		host = srv.srv.Listener.Addr().String()

		args = []string{host, containerId}
	})

	AfterEach(func() {
		srv.srv.CloseClientConnections()
		srv.srv.Close()
	})

	It("implements cli.Command", func() {
		var _ cli.Command = cmd
	})

	It("attaches to the remote shell", func() {
		go func() {
			defer GinkgoRecover()
			cmd.Run(args)
			Expect(ui.ErrorWriter.(*bytes.Buffer).String()).To(BeEmpty())
		}()

		Eventually(func() bool {
			return srv.resized
		}).Should(BeTrue())

		Eventually(func() bool {
			return srv.finished
		}).Should(BeTrue())
	})
})
