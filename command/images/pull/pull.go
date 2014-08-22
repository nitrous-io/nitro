package pull

import (
	"fmt"
	"io"

	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/freighter/freighterclient"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/images"
	"github.com/nitrous-io/nitro/pkg/jsonstream"
	"github.com/nitrous-io/nitro/pkg/nitrousapi"
	"github.com/nitrous-io/nitrous-common/httpclient"
	flag "github.com/ogier/pflag"
)

const (
	Usage = `
Usage: nitro images:pull <host> [registry_host/][username/]<repo_name:tag>

  pull image onto a given host
`
)

type Response struct {
	Image images.Image `json:"image"`
}

type Command struct {
	UI     *cli.BasicUi
	Server string
	Client *httpclient.Client
}

func init() {
	command.RegisterAuth("images:pull", NewCommand)
}

func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI: &cli.BasicUi{
			Reader: os.Stdin,
			Writer: os.Stdout,
		},
		Server: command.Env.APIURL,
		Client: httpclient.New(token),
	}, nil
}

func (c *Command) Help() string {
	return strings.TrimSpace(Usage)
}

func (c *Command) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("images:pull", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 2 {
		c.UI.Output(c.Help())
		return 1
	}

	stdout, ok := c.UI.Writer.(*os.File)
	if !ok {
		c.UI.Error("Requires a working terminal")
		return 1
	}

	host, image := args[0], args[1]

	h, err := nitrousapi.GetHost(c.Client, c.Server, host)
	if err != nil {
		c.UI.Error(fmt.Sprintf("failed to get host %s", host))
		return 1
	}

	client := freighterclient.NewClient(freighterclient.NewHost(
		h.IPAddress,
		int(h.FreighterdPort),
		h.Domain,
	))

	c.UI.Output(fmt.Sprintf("Pulling image '%s' to '%s'...", image, host))

	reader, writer := io.Pipe()

	// decode and write the json stream output to stdout
	go jsonstream.DisplayJSONMessagesStream(reader, stdout, stdout.Fd(), true)

	_, err = client.PullImage(image, writer)
	if err != nil {
		c.UI.Error(fmt.Sprintf("failed to pull image %s on host %s", image, host))
		return 1
	}
	return 0
}

func (c *Command) Synopsis() string {
	return "Pull image onto a host"
}
