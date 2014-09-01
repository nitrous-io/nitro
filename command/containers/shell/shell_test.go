package shell

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/kr/pty"
	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/pkg/wsconn"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestShell(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "shell")
}

var _ = Describe("Command", func() {
	It("requires container slug", func() {
		cmd := &Command{
			UI: &cli.BasicUi{ErrorWriter: bytes.NewBuffer(nil)},
		}
		code := cmd.Run([]string{})
		Expect(code).To(Equal(1), "Requires container slug")
	})

	It("attaches to the remote shell", func() {
		var hostID int64 = 1
		containerID := "container-slug"

		ns := newNitrousServer(containerID, "shell-id", hostID)
		defer ns.Close()

		stdin, stdout, _ := pty.Open()
		ew := bytes.NewBuffer(nil)
		ui := &cli.BasicUi{
			Reader:      stdin,
			Writer:      stdout,
			ErrorWriter: ew,
		}
		cmd := &Command{
			UI:     ui,
			server: ns.srv.URL,
			client: httpclient.New("authtoken"), // FIXME This doesn't do anything.
		}

		go func() {
			defer GinkgoRecover()
			cmd.Run([]string{containerID})
		}()

		Consistently(ew.String()).Should(BeEmpty())
		Eventually(ns.ts.resized).Should(Receive())
		Eventually(ns.ts.finished).Should(BeClosed())
	})

	Describe("getLoginCommand", func() {
		It("returns the logins command", func() {
			testCases := [][]string{
				{"", "/sbin/getty -a root tty"},
				{"action", "/sbin/getty -a action tty"},
			}

			for _, tc := range testCases {
				cmd := getLoginCommand(tc[0])
				Expect(cmd).To(Equal(tc[1]), "Incorrect login command")
			}
		})
	})
})

// nitrous test server
type nitrousServer struct {
	srv         *httptest.Server
	ts          *testServer
	containerID string
	hostID      int64
}

func (s *nitrousServer) getContainer(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"CID":     s.containerID,
		"host_id": s.hostID,
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"container": resp,
	})
}

func (s *nitrousServer) getHost(w http.ResponseWriter, r *http.Request) {
	addr := s.ts.srv.Listener.Addr().String()
	host, port, _ := net.SplitHostPort(addr)
	portInt, _ := strconv.ParseInt(port, 10, 64)
	resp := map[string]interface{}{
		"ip_address":      host,
		"freighterd_port": portInt,
		"domain":          "example.com",
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"host": resp,
	})
}

func (s *nitrousServer) Close() {
	s.ts.Close()
	s.srv.CloseClientConnections()
	s.srv.Close()
}

func newNitrousServer(containerID, shellID string, hostID int64) *nitrousServer {
	ts := newTestServer(containerID, shellID)
	s := &nitrousServer{
		ts:          ts,
		containerID: containerID,
		hostID:      hostID,
	}
	r := mux.NewRouter()
	r.HandleFunc(fmt.Sprintf("/api/v0/containers/%s", containerID), s.getContainer).Methods("GET")
	r.HandleFunc(fmt.Sprintf("/api/v0/hosts/%d", hostID), s.getHost).Methods("GET")
	srv := httptest.NewServer(r)
	s.srv = srv
	return s
}

// freighter test server
type testServer struct {
	srv *httptest.Server

	containerID string
	hostID      int64
	shellID     string

	stdin, stdout *bytes.Buffer
	resized       chan struct{}
	finished      chan struct{}
}

func (s *testServer) createShell(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	Expect(id).To(Equal(s.containerID))
	json.NewEncoder(w).Encode(map[string]string{
		"id": s.shellID,
	})
}

func (s *testServer) attachShell(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	Expect(id).To(Equal(s.shellID))

	upgrader := websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	stream := wsconn.NewConnection(conn)
	defer stream.Close()
	go io.Copy(s.stdin, stream)
	io.Copy(stream, s.stdout)
	close(s.finished)
}

func (s *testServer) resizeShell(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	Expect(id).To(Equal(s.shellID))
	s.resized <- struct{}{}
	w.WriteHeader(http.StatusAccepted)
}

func (s *testServer) Close() {
	s.srv.CloseClientConnections()
	s.srv.Close()
}

func newTestServer(containerID, shellID string) *testServer {
	s := &testServer{
		containerID: containerID,
		shellID:     shellID,
		stdin:       bytes.NewBuffer(nil),
		stdout:      bytes.NewBuffer(nil),
		resized:     make(chan struct{}, 1),
		finished:    make(chan struct{}),
	}
	r := mux.NewRouter()
	r.HandleFunc("/containers/{id}/shells", s.createShell).Methods("POST")
	r.HandleFunc("/shells/{id}/attach/ws", s.attachShell).Methods("GET")
	r.HandleFunc("/shells/{id}/resize", s.resizeShell).Methods("POST")

	srv := httptest.NewServer(r)
	s.srv = srv
	return s
}
