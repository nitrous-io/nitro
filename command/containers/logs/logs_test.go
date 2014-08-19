package logs_test

import (
	"bytes"
	"strconv"

	"net"
	"net/http"

	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/containers/logs"
	"github.com/nitrous-io/nitrous-common/httpclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func TestLogs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "logs")
}

var _ = Describe("logs", func() {
	var (
		cmd *logs.Command

		stdout *bytes.Buffer
		// Nitrous Server
		nsrv      *ghttp.Server
		nhandlers http.HandlerFunc
		hhandlers http.HandlerFunc
		// Freighter Server
		fsrv      *ghttp.Server
		fhandlers http.HandlerFunc

		nStatusCode   int
		nResponseBody interface{}

		hStatusCode   int
		hResponseBody interface{}

		fStatusCode   int
		fResponseBody string
	)

	BeforeEach(func() {
		// Setup handlers for nitrous server (containers)
		nhandlers = ghttp.CombineHandlers(
			ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
			ghttp.VerifyRequest("GET", "/api/v0/containers/foo-container-42"),
			ghttp.RespondWithJSONEncodedPtr(&nStatusCode, &nResponseBody),
		)
		// Setup handlers for nitrous server (hosts)
		hhandlers = ghttp.CombineHandlers(
			ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
			ghttp.VerifyRequest("GET", "/api/v0/hosts/1"),
			ghttp.RespondWithJSONEncodedPtr(&hStatusCode, &hResponseBody),
		)
		nsrv = ghttp.NewServer()
		nsrv.AppendHandlers(nhandlers, hhandlers)

		// Setup handlers for freighter which nitro
		// will be querying to get the log stream
		fhandlers = ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", "/containers/42/logs"),
			ghttp.RespondWithPtr(&fStatusCode, &fResponseBody),
		)
		fsrv = ghttp.NewTLSServer()
		fsrv.AppendHandlers(fhandlers)

		stdout = bytes.NewBuffer(nil)

		cmd = &logs.Command{
			UI: &command.BasicUI{cli.BasicUi{
				Writer:      stdout,
				ErrorWriter: bytes.NewBuffer(nil),
			}},
			Server: nsrv.URL(),
			Client: httpclient.New("correct_token"),
		}
	})

	AfterEach(func() {
		nsrv.Close()
		fsrv.Close()
	})

	Context("with valid parameters", func() {
		BeforeEach(func() {
			addr := fsrv.HTTPTestServer.Listener.Addr().String()
			host, port, _ := net.SplitHostPort(addr)
			portInt, err := strconv.ParseInt(port, 10, 64)
			Expect(err).To(BeNil())

			nStatusCode = http.StatusOK
			nResponseBody = map[string]interface{}{
				"container": map[string]interface{}{
					"CID":     "42",
					"host_id": 1,
				},
			}

			hStatusCode = http.StatusOK
			hResponseBody = map[string]interface{}{
				"host": map[string]interface{}{
					"ip_address":      host,
					"freighterd_port": portInt,
					"domain":          "example.com",
					"ca":              localhostCert,
				},
			}

			fStatusCode = http.StatusOK
			fResponseBody = "hello im a log"
		})

		It("get logs of the container", func() {
			code := cmd.Run([]string{"foo-container-42"})
			Expect(code).To(Equal(0))
			Expect(stdout.String()).To(Equal("hello im a log"))
		})
	})

	Context("without providing container slug", func() {
		It("requires container slug", func() {
			code := cmd.Run([]string{})
			Expect(code).To(Equal(1), "Requires container slug")
		})
	})
})

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
