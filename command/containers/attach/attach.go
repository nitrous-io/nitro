package attach

import (
	"fmt"
	"net/url"

	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/containers"
	"github.com/nitrous-io/nitro/pkg/nitrousapi"
	"github.com/nitrous-io/nitro/pkg/protoshell"
	"github.com/nitrous-io/nitrous-common/httpclient"
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

	stdin, ok := c.UI.Reader.(*os.File)
	if !ok {
		c.UI.Error("Requires a working terminal")
		return 1
	}
	stdout, ok := c.UI.Writer.(*os.File)
	if !ok {
		c.UI.Error("Requires a working terminal")
		return 1
	}

	// construct params header that includes the wanted command
	params := url.Values{}
	params.Set("token", token)
	params.Set("binary", "true")
	endpointURI := fmt.Sprintf("wss://%s/containers/%s/attach?%s", host, cn.CID, params.Encode())
	sh := protoshell.New(endpointURI, tlsConfig)
	if err := sh.Run(stdin.Fd(), stdout.Fd(), stdin, stdout); err != nil {
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
