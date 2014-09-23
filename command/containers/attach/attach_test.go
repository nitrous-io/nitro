package attach_test

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command/containers/attach"
	"github.com/nitrous-io/nitro/consts"
	"github.com/nitrous-io/nitrous-common/httpclient"
	"github.com/nitrous-io/nitrous-common/wsconn"

	"github.com/nitrous-io/nitro/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func TestAttach(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "containers/attach")
}

var _ = Describe("containers/attach", func() {
	var (
		// Nitrous Server
		nsrv *ghttp.Server
		// Freighter Server
		fsrv *ghttp.Server
	)

	BeforeEach(func() {
		// Setup Freighter server.
		fsrv = ghttp.NewServer()
		var err error
		fsrv.HTTPTestServer, err = test.NewTLSServer(fsrv, test.SampleHostname, []byte(consts.CACert))
		Expect(err).To(BeNil())
		// Setup handlers: attach to container.
		fsrv.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", "/containers/deadbeef/attach", "token=container-token"),
			func(w http.ResponseWriter, r *http.Request) {
				upgrader := websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}
				conn, err := upgrader.Upgrade(w, r, nil)
				Expect(err).To(BeNil())
				stream := wsconn.NewConnection(conn)
				defer stream.Close()

				stream.Write([]byte("d|attach output"))
				// Write a heartbeat message to see if it is ignored
				stream.Write([]byte("h"))
				// Send a close message
				stream.Write([]byte("c|connection closed"))
			},
		))

		// Setup nitrous server.
		nsrv = ghttp.NewServer()
		// Setup handlers: request container token.
		nsrv.AppendHandlers(ghttp.CombineHandlers(
			func(w http.ResponseWriter, r *http.Request) {
				r.ParseForm()
				Expect(r.PostForm["permission[]"]).To(Equal([]string{"attach"}))
			},
			ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
			ghttp.VerifyRequest("POST", "/api/v0/containers/foo-container-42/token"),
			ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]string{"token": "container-token"}),
		))
		// Setup handlers: get container info.
		nsrv.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
			ghttp.VerifyRequest("GET", "/api/v0/containers/foo-container-42"),
			ghttp.RespondWithJSONEncoded(
				http.StatusOK,
				map[string]interface{}{
					"container": map[string]interface{}{
						"CID":     "deadbeef",
						"host_id": 1,
						"state":   "running",
					},
				}),
		))
		// Setup handlers: get host info.
		addr := fsrv.HTTPTestServer.Listener.Addr().String()
		fhost, port, _ := net.SplitHostPort(addr)
		fport, err := strconv.ParseInt(port, 10, 64)
		Expect(err).To(BeNil())
		nsrv.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
			ghttp.VerifyRequest("GET", "/api/v0/hosts/1"),
			ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]interface{}{
				"host": map[string]interface{}{
					"ip_address":      fhost,
					"freighterd_port": fport,
					"hostname":        test.SampleHostname,
				},
			}),
		))
	})

	AfterEach(func() {
		nsrv.Close()
		fsrv.Close()
	})

	It("requires container slug", func() {
		cmd := &attach.Command{
			UI: &cli.BasicUi{ErrorWriter: bytes.NewBuffer(nil)},
		}
		code := cmd.Run([]string{})
		Expect(code).To(Equal(1))
	})

	It("attaches to the container", func(done Done) {
		stdoutReader, stdout := io.Pipe()
		cmd := &attach.Command{
			UI: &cli.BasicUi{
				Reader:      bytes.NewBuffer(nil),
				Writer:      stdout,
				ErrorWriter: bytes.NewBuffer(nil),
			},
			Server: nsrv.URL(),
			Client: httpclient.New("correct_token"),
		}

		exited := make(chan struct{})
		go func() {
			cmd.Run([]string{"foo-container-42"})
			close(exited)
		}()

		b := make([]byte, len("attach output"))
		_, err := stdoutReader.Read(b)
		Expect(err).To(BeNil())
		Expect(string(b)).To(Equal("attach output"))

		// Close pipe so that the command will exit.
		stdout.Close()
		stdoutReader.Close()

		<-exited
		close(done)

		Expect(nsrv.ReceivedRequests()).To(HaveLen(3))
		Expect(fsrv.ReceivedRequests()).To(HaveLen(1))
	})
})
