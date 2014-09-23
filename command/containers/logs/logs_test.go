package logs_test

import (
	"bytes"
	"io"
	"strconv"

	"net"
	"net/http"

	"testing"

	"github.com/gorilla/websocket"
	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/containers/logs"
	"github.com/nitrous-io/nitro/consts"
	"github.com/nitrous-io/nitro/test"
	"github.com/nitrous-io/nitrous-common/httpclient"
	"github.com/nitrous-io/nitrous-common/wsconn"
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
		nsrv *ghttp.Server
		// Freighter Server
		fsrv *ghttp.Server

		nStatusCode   int
		nResponseBody interface{}

		hStatusCode   int
		hResponseBody interface{}

		fStatusCode   int
		fResponseBody string
	)

	BeforeEach(func() {
		// Setup handlers for nitrous server (request container token).
		handlers1 := ghttp.CombineHandlers(
			func(w http.ResponseWriter, r *http.Request) {
				r.ParseForm()
				Expect(r.PostForm["permission[]"]).To(Equal([]string{"logs"}))
			},
			ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
			ghttp.VerifyRequest("POST", "/api/v0/containers/foo-container-42/token"),
			ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]string{"token": "container-token"}),
		)
		// Setup handlers for nitrous server (get container info).
		handlers2 := ghttp.CombineHandlers(
			ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
			ghttp.VerifyRequest("GET", "/api/v0/containers/foo-container-42"),
			ghttp.RespondWithJSONEncodedPtr(&nStatusCode, &nResponseBody),
		)
		// Setup handlers for nitrous server (get host info).
		handlers3 := ghttp.CombineHandlers(
			ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
			ghttp.VerifyRequest("GET", "/api/v0/hosts/1"),
			ghttp.RespondWithJSONEncodedPtr(&hStatusCode, &hResponseBody),
		)
		nsrv = ghttp.NewServer()
		nsrv.AppendHandlers(handlers1, handlers2, handlers3)

		// Setup handlers for freighter which nitro
		// will be querying to get the log stream
		fsrv = ghttp.NewServer()
		var err error
		fsrv.HTTPTestServer, err = test.NewTLSServer(fsrv, test.SampleHostname, []byte(consts.CACert))
		Expect(err).To(BeNil())
		// Setup handlers: attach to container.
		fsrv.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest(
				"GET",
				"/containers/deadbeef/logs",
				"follow=false&tail=all&timestamps=false&token=container-token"),
			func(w http.ResponseWriter, r *http.Request) {
				upgrader := websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}
				conn, err := upgrader.Upgrade(w, r, nil)
				Expect(err).To(BeNil())
				stream := wsconn.NewConnection(conn)
				defer stream.Close()

				stream.Write([]byte("d|hello im a log"))
				// Write a heartbeat message to see if it is ignored
				stream.Write([]byte("h"))
				// Send a close message
				stream.Write([]byte("c|connection closed"))
			},
		))

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
					"CID":     "deadbeef",
					"host_id": 1,
				},
			}

			hStatusCode = http.StatusOK
			hResponseBody = map[string]interface{}{
				"host": map[string]interface{}{
					"ip_address":      host,
					"hostname":        test.SampleHostname,
					"freighterd_port": portInt,
				},
			}

			fStatusCode = http.StatusOK
			fResponseBody = "hello im a log"
		})

		It("get logs of the container", func(done Done) {
			stdoutReader, stdout := io.Pipe()
			cmd = &logs.Command{
				UI: &command.BasicUI{cli.BasicUi{
					Writer:      stdout,
					ErrorWriter: bytes.NewBuffer(nil),
				}},
				Server: nsrv.URL(),
				Client: httpclient.New("correct_token"),
			}

			exited := make(chan struct{})
			go func() {
				cmd.Run([]string{"foo-container-42"})
				close(exited)
			}()

			b := make([]byte, len("hello im a log"))
			_, err := stdoutReader.Read(b)
			Expect(err).To(BeNil())
			Expect(string(b)).To(Equal("hello im a log"))

			// Close pipe so that the command will exit.
			stdout.Close()
			stdoutReader.Close()

			<-exited
			close(done)
		})
	})

	Context("without providing container slug", func() {
		It("requires container slug", func() {
			cmd = &logs.Command{
				UI: &command.BasicUI{cli.BasicUi{
					Writer:      bytes.NewBuffer(nil),
					ErrorWriter: bytes.NewBuffer(nil),
				}},
				Server: nsrv.URL(),
				Client: httpclient.New("correct_token"),
			}

			code := cmd.Run([]string{})
			Expect(code).To(Equal(1), "Requires container slug")
		})
	})
})
