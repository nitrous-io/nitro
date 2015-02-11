package shell

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/kr/pty"
	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/consts"
	"github.com/nitrous-io/nitrous-common/httpclient"
	"github.com/nitrous-io/nitrous-common/wsconn"

	"github.com/nitrous-io/nitro/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestShell(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "containers/shell")
}

var _ = Describe("Command", func() {
	It("requires container slug", func() {
		cmd := &Command{
			UI: &cli.BasicUi{ErrorWriter: bytes.NewBuffer(nil)},
		}
		code := cmd.Run([]string{})
		Expect(code).To(Equal(1), "Requires container slug")
	})

	It("requires a running container", func() {
		containerID := "container-slug"
		ns := newNitrousServer(containerID, "container-token", "shell-id", 1)
		ns.containerState = "stopped"
		defer ns.Close()

		ew := bytes.NewBuffer(nil)
		cmd := &Command{
			UI:     &cli.BasicUi{ErrorWriter: ew},
			server: ns.srv.URL,
			client: httpclient.New("authtoken"), // FIXME This doesn't do anything.

		}
		code := cmd.Run([]string{containerID})
		Expect(ew.String()).To(ContainSubstring("Requires a running container"))
		Expect(code).To(Equal(1))
	})

	It("opens a shell to running container", func() {
		var hostID int64 = 1
		containerID := "container-slug"

		ns := newNitrousServer(containerID, "container-token", "shell-id", hostID)
		defer ns.Close()

		stdin, stdout, _ := pty.Open()
		ew := bytes.NewBuffer(nil)
		ui := &cli.BasicUi{
			Reader:      stdin,
			Writer:      stdout,
			ErrorWriter: ew,
		}
		tlsConf, err := command.ClientConfig(test.SampleHostname)
		Expect(err).To(BeNil(), "failed to generate client tls config")
		cmd := &Command{
			UI:        ui,
			server:    ns.srv.URL,
			client:    httpclient.New("authtoken"), // FIXME This doesn't do anything.
			tlsConfig: tlsConf,
		}

		exited := make(chan struct{})
		go func() {
			cmd.Run([]string{containerID})
			close(exited)
		}()

		<-exited
		Expect(ew.String()).To(BeEmpty())

		Eventually(ns.ts.finished).Should(BeClosed())
	})

	Describe("getLoginCommand", func() {
		It("returns the logins command", func() {
			testCases := [][]string{
				{"", "xterm-256color", "script -q -c '/bin/login -f root -p TERM=xterm-256color' /dev/null || /bin/login -f root -p TERM=xterm-256color"},
				{"action", "xterm", "script -q -c '/bin/login -f action -p TERM=xterm' /dev/null || /bin/login -f action -p TERM=xterm"},
			}

			for _, tc := range testCases {
				user := tc[0]
				term := tc[1]
				command := tc[2]

				cmd, err := getLoginCommand(user, term)
				Expect(err).To(BeNil())
				Expect(cmd).To(Equal(command), "Incorrect login command")
			}
		})

		It("disallows invalid usernames", func() {
			testCases := []string{
				"$(touch /tmp/ohnoes)root",
				"`touch /tmp/ohnoes`action",
				`Administrator`,
				`mismatched double quotes"`,
				strings.Repeat("a", 33),
			}

			for _, tc := range testCases {
				_, err := getLoginCommand(tc, "")
				Expect(err).To(MatchError("invalid username"))
			}
		})
	})
})

// nitrous test server
type nitrousServer struct {
	srv            *httptest.Server
	ts             *testServer
	containerID    string
	hostID         int64
	containerState string
	containerToken string
}

func (s *nitrousServer) getContainer(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"CID":     s.containerID,
		"host_id": s.hostID,
		"state":   s.containerState,
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"container": resp,
	})
}

func (s *nitrousServer) getContainerToken(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"token": s.containerToken})
}

func (s *nitrousServer) getHost(w http.ResponseWriter, r *http.Request) {
	addr := s.ts.srv.Listener.Addr().String()
	host, port, _ := net.SplitHostPort(addr)
	portInt, _ := strconv.ParseInt(port, 10, 64)
	resp := map[string]interface{}{
		"ip_address":      host,
		"proxy_hostname":  host,
		"freighterd_port": portInt,
		"hostname":        test.SampleHostname,
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"host": resp,
	})

	// switch command freighterd port to the custom port for duration of
	// the request
	command.FreighterdPort = int(portInt)
}

func (s *nitrousServer) Close() {
	s.ts.Close()
	s.srv.CloseClientConnections()
	s.srv.Close()
	command.FreighterdPort = 65333
}

func newNitrousServer(containerID, containerToken, shellID string, hostID int64) *nitrousServer {
	ts := newTestServer(containerID, shellID, containerToken)
	s := &nitrousServer{
		ts:             ts,
		containerID:    containerID,
		hostID:         hostID,
		containerState: "running",
		containerToken: containerToken,
	}
	r := mux.NewRouter()
	r.HandleFunc(fmt.Sprintf("/api/v0/containers/%s", containerID), s.getContainer).Methods("GET")
	r.HandleFunc(fmt.Sprintf("/api/v0/containers/%s/token", containerID), s.getContainerToken).Methods("POST")
	r.HandleFunc(fmt.Sprintf("/api/v0/hosts/%d", hostID), s.getHost).Methods("GET")
	srv := httptest.NewServer(r)
	s.srv = srv
	return s
}

// freighter test server
type testServer struct {
	srv *httptest.Server

	containerID    string
	containerToken string

	stdin, stdout *bytes.Buffer
	// signal that the shell is called and finished
	finished chan struct{}
}

func (s *testServer) newShell(w http.ResponseWriter, r *http.Request) {
	Expect(r.FormValue("token")).To(Equal(s.containerToken))
	id := mux.Vars(r)["id"]
	Expect(id).To(Equal(s.containerID))

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

func (s *testServer) Close() {
	s.srv.CloseClientConnections()
	s.srv.Close()
}

func newTestServer(containerID, shellID, containerToken string) *testServer {
	s := &testServer{
		containerID:    containerID,
		containerToken: containerToken,
		stdin:          bytes.NewBuffer(nil),
		stdout:         bytes.NewBuffer(nil),
		finished:       make(chan struct{}),
	}
	r := mux.NewRouter()
	r.HandleFunc("/containers/{id}/shell", s.newShell).Methods("GET")

	srv := httptest.NewServer(r)
	tlsSrv, err := test.NewTLSServer(srv.Config.Handler, test.SampleHostname, consts.CACert)
	Expect(err).To(BeNil(), "failed to start tls server")
	s.srv = tlsSrv
	return s
}
