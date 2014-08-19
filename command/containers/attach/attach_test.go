package attach

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
	"github.com/kr/pty"
	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAttach(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "containers/attach")
}

var _ = Describe("Command", func() {
	It("requires container slug", func() {
		cmd := &Command{
			UI: &cli.BasicUi{ErrorWriter: bytes.NewBuffer(nil)},
		}
		code := cmd.Run([]string{})
		Expect(code).To(Equal(1), "Requires container slug", 1)
	})

	It("attaches to the container", func() {
		containerID := "container-slug"
		hostID := 1
		ns := newNitrousServer(containerID, int64(hostID))
		defer ns.Close()

		stdin, stdout, _ := pty.Open()
		ui := &cli.BasicUi{
			Reader:      stdin,
			Writer:      stdout,
			ErrorWriter: bytes.NewBuffer(nil),
		}
		cmd := &Command{
			UI:     ui,
			Server: ns.srv.URL,
			Client: httpclient.New("authtoken"),
		}

		go func() {
			defer GinkgoRecover()
			cmd.Run([]string{containerID})
			Expect(ui.ErrorWriter.(*bytes.Buffer).String()).To(BeEmpty())
		}()

		Eventually(func() bool {
			return ns.ts.finished
		}).Should(BeTrue())
	})
})

// nitrous test server
type nitrousServer struct {
	srv         *httptest.Server
	ts          *testServer
	containerID string
	hostID      int64
}

func (s *nitrousServer) getHost(w http.ResponseWriter, r *http.Request) {
	addr := s.ts.srv.Listener.Addr().String()
	host, port, _ := net.SplitHostPort(addr)
	portInt, _ := strconv.ParseInt(port, 10, 64)
	resp := map[string]interface{}{
		"ip_address":      host,
		"freighterd_port": portInt,
		"domain":          "example.com",
		"ca":              localhostCert,
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"host": resp,
	})
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

func (s *nitrousServer) Close() {
	s.ts.Close()
	s.srv.CloseClientConnections()
	s.srv.Close()
}

func newNitrousServer(containerID string, hostID int64) *nitrousServer {
	ts := newTestServer(containerID)
	s := &nitrousServer{
		ts:          ts,
		containerID: containerID,
		hostID:      hostID,
	}
	r := mux.NewRouter()
	r.HandleFunc("/api/v0/containers/"+containerID, s.getContainer).Methods("GET")
	r.HandleFunc("/api/v0/hosts/"+fmt.Sprintf("%d", hostID), s.getHost).Methods("GET")
	srv := httptest.NewServer(r)
	s.srv = srv
	return s
}

// freighter test server
type testServer struct {
	srv *httptest.Server

	containerID string

	stdin, stdout *bytes.Buffer
	finished      bool
}

func (s *testServer) attachToContainer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	Expect(id).To(Equal(s.containerID))
	stdin, stdout, _ := hijackServer(w)
	go io.Copy(s.stdin, stdin)
	io.Copy(stdout, s.stdout)
	s.finished = true
}

func (s *testServer) Close() {
	s.srv.CloseClientConnections()
	s.srv.Close()
}

func newTestServer(containerID string) *testServer {
	s := &testServer{
		containerID: containerID,
		stdin:       bytes.NewBuffer(nil),
		stdout:      bytes.NewBuffer(nil),
	}
	r := mux.NewRouter()
	r.HandleFunc("/containers/{id}/attach", s.attachToContainer).Methods("POST")

	srv := httptest.NewTLSServer(r)
	s.srv = srv
	return s
}

func hijackServer(w http.ResponseWriter) (io.ReadCloser, io.Writer, error) {
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		return nil, nil, err
	}
	// Flush the options to make sure the client sets the raw mode
	fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Type: application/vnd.containerd.raw-stream\r\n\r\n")
	return conn, conn, nil
}

// These certs are the same as the certs embeded with net/http/httptest.
// If there's mismatched certs error, look at net/http/httptest/server.go to
// check if the cert has changed.
//
// localhostCert is a PEM-encoded TLS cert with SAN IPs
// "127.0.0.1" and "[::1]", expiring at the last second of 2049 (the end
// of ASN.1 time).
// generated from src/pkg/crypto/tls:
// go run generate_cert.go  --rsa-bits 512 --host 127.0.0.1,::1,example.com --ca --start-date "Jan 1 00:00:00 1970" --duration=1000000h
var localhostCert = `-----BEGIN CERTIFICATE-----
MIIBdzCCASOgAwIBAgIBADALBgkqhkiG9w0BAQUwEjEQMA4GA1UEChMHQWNtZSBD
bzAeFw03MDAxMDEwMDAwMDBaFw00OTEyMzEyMzU5NTlaMBIxEDAOBgNVBAoTB0Fj
bWUgQ28wWjALBgkqhkiG9w0BAQEDSwAwSAJBAN55NcYKZeInyTuhcCwFMhDHCmwa
IUSdtXdcbItRB/yfXGBhiex00IaLXQnSU+QZPRZWYqeTEbFSgihqi1PUDy8CAwEA
AaNoMGYwDgYDVR0PAQH/BAQDAgCkMBMGA1UdJQQMMAoGCCsGAQUFBwMBMA8GA1Ud
EwEB/wQFMAMBAf8wLgYDVR0RBCcwJYILZXhhbXBsZS5jb22HBH8AAAGHEAAAAAAA
AAAAAAAAAAAAAAEwCwYJKoZIhvcNAQEFA0EAAoQn/ytgqpiLcZu9XKbCJsJcvkgk
Se6AbGXgSlq+ZCEVo0qIwSgeBqmsJxUu7NCSOwVJLYNEBO2DtIxoYVk+MA==
-----END CERTIFICATE-----`
