package run

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/pkg/httpshell"
	"github.com/nitrous-io/nitro/pkg/nitrousapi"
	"github.com/nitrous-io/nitrous-common/httpclient"
)

const usage = `
Usage: nitro run [container] [cmd]

run a command on a remote docker container.
`

func init() {
	command.RegisterAuth("run", NewCommand)
}

type Command struct {
	UI     *cli.BasicUi
	client *httpclient.Client
	server string
}

func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI: &cli.BasicUi{
			Reader: os.Stdin,
			Writer: os.Stdout,
		},
		server: command.Env.APIURL,
		client: httpclient.New(token),
	}, nil
}

func (c *Command) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("run", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) < 2 {
		c.UI.Error(c.Help())
		return 1
	}

	slug := args[0]
	cr, err := nitrousapi.GetContainer(c.client, c.server, slug)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	host := fmt.Sprintf("%s:%d", cr.Host.IPAddress, cr.Host.FreighterdPort)
	command := strings.Join(args[1:], " ")

	stdin := c.UI.Reader
	stdout := c.UI.Writer
	pty, ok := stdout.(*os.File)
	if !ok {
		c.UI.Error("Requires a working terminal")
		return 1
	}

	err = httpshell.Run(host, cr.CID, command, pty.Fd(), stdin, stdout)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Exited: %v", err))
		return 1
	}
	return 0
}

func (c *Command) Help() string {
	return strings.TrimSpace(usage)
}

func (c *Command) Synopsis() string {
	return "Run a command on a remote box"
}
