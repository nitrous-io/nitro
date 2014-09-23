package attach

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/containers"
	"github.com/nitrous-io/nitro/pkg/nitrousapi"
	"github.com/nitrous-io/nitrous-common/httpclient"
	"github.com/nitrous-io/nitrous-common/wsconn"
)

var usage = `
Usage: nitro attach <container slug>

  attach to a running container
`

type Command struct {
	UI     *cli.BasicUi
	Client *httpclient.Client
	Server string
}

func init() {
	command.RegisterAuth("attach", NewCommand)
}

func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI: &cli.BasicUi{
			Reader:      os.Stdin,
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		},
		Server: command.Env.APIURL,
		Client: httpclient.New(token),
	}, nil
}

func (c *Command) Run(args []string) int {
	if len(args) != 1 {
		c.UI.Error(c.Help())
		return 1
	}
	slug := args[0]

	token, err := containers.GetToken(slug, c.Server, []string{"attach"}, c.Client)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	cn, err := nitrousapi.GetContainer(c.Client, c.Server, slug)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}
	if cn.State != "running" {
		c.UI.Error("Requires a running container")
		return 1
	}

	host := fmt.Sprintf("%s:%d", cn.Host.IPAddress, cn.Host.FreighterdPort)
	tlsConfig, err := command.ClientConfig(cn.Host.Hostname)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error establishing secure connection to %s", host))
		return 1
	}

	if err := c.attach(host, token, cn.CID, tlsConfig, c.UI.Reader, c.UI.Writer); err != nil {
		c.UI.Error(fmt.Sprintf("Exited: %v", err))
		return 1
	}
	return 0
}

func (c *Command) Help() string {
	return strings.TrimSpace(usage)
}

func (c *Command) Synopsis() string {
	return "Attach to a running container"
}

func (c *Command) attach(host, token, containerID string, tlsConfig *tls.Config, stdin io.Reader, stdout io.Writer) error {
	params := url.Values{}
	params.Set("token", token)
	wsEndpointURI := fmt.Sprintf("wss://%s/containers/%s/attach?%s", host, containerID, params.Encode())
	dialer := websocket.Dialer{}
	if tlsConfig != nil {
		dialer.TLSClientConfig = tlsConfig
	}
	ws, _, err := dialer.Dial(wsEndpointURI, http.Header{})
	if err != nil {
		return errors.New("Failed to attach to container.")
	}

	conn := wsconn.NewConnection(ws)
	defer conn.Close()

	return containers.CopyFromWebsocket(stdout, conn)
}
