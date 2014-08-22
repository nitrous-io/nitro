package pull_test

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"

	"github.com/kr/pty"
	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command/images/pull"
	"github.com/nitrous-io/nitrous-common/httpclient"
	"github.com/onsi/gomega/ghttp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "images/pull")
}

var _ = Describe("images/pull", func() {
	It("outputs formatted download progress", func() {
		pullResponse := `
		{"status":"Pulling..."}
		{"status":"Pulling", "progress":"1 B/ 100 B", "progressDetail":{"current":1, "total":100}}
		`
		freighterSrv := setupFreighter(http.StatusOK, pullResponse)
		nitrousSrv := setupNitrous("myhost", freighterSrv)
		defer freighterSrv.Close()
		defer nitrousSrv.Close()

		stdin, stdout, _ := pty.Open()
		ui := &cli.BasicUi{
			Reader:      stdin,
			Writer:      stdout,
			ErrorWriter: bytes.NewBuffer(nil),
		}
		cmd := &pull.Command{
			UI:     ui,
			Server: nitrousSrv.URL(),
			Client: httpclient.New("correct_token"),
		}

		buf := bytes.NewBuffer(nil)
		var w sync.WaitGroup
		w.Add(1)
		go func() {
			io.Copy(buf, stdin)
			w.Done()
		}()

		cmd.Run([]string{"myhost", "scratch"})
		stdout.Close()
		w.Wait()
		Expect(buf.String()).To(ContainSubstring("Pulling image 'scratch' to 'myhost'..."))
		Expect(buf.String()).To(ContainSubstring("Pulling..."))
	})

	Context("when given wrong arguments", func() {
		It("prints usage and exits with error", func() {
			stdout := bytes.NewBuffer(nil)
			ui := &cli.BasicUi{Writer: stdout}
			cmd := &pull.Command{UI: ui}
			code := cmd.Run([]string{})
			Expect(stdout.String()).To(ContainSubstring("Usage:"))
			Expect(code).To(Equal(1))
		})
	})

	Context("Help", func() {
		It("outputs usage", func() {
			cmd := &pull.Command{}
			Expect(cmd.Help()).To(ContainSubstring("Usage: nitro images:pull"))
		})
	})
})

func setupNitrous(slug string, freighterSrv *ghttp.Server) *ghttp.Server {
	addr := freighterSrv.HTTPTestServer.Listener.Addr().String()
	host, port, _ := net.SplitHostPort(addr)
	portInt, _ := strconv.ParseInt(port, 10, 64)
	resp := map[string]interface{}{
		"ip_address":      host,
		"freighterd_port": portInt,
		"domain":          "example.com",
	}
	// Setup handlers for nitrous server (hosts)
	handler := ghttp.CombineHandlers(
		ghttp.VerifyHeaderKV("Authorization", "Bearer correct_token"),
		ghttp.VerifyRequest("GET", "/api/v0/hosts/"+slug),
		ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]interface{}{
			"host": resp,
		}),
	)
	srv := ghttp.NewServer()
	srv.AppendHandlers(handler)
	return srv
}

func setupFreighter(statusCode int, response string) *ghttp.Server {
	// Setup handlers for nitrous server (hosts)
	handler := ghttp.CombineHandlers(
		ghttp.VerifyRequest("POST", "/images/pull"),
		ghttp.RespondWith(statusCode, response),
	)
	srv := ghttp.NewServer()
	srv.AppendHandlers(handler)
	return srv
}
