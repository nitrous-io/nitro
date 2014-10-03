package stop

import (
	"fmt"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitrous-common/httpclient"
	flag "github.com/ogier/pflag"

	. "github.com/nitrous-io/nitro/command/containers"
)

const yesUsage = "Skip confirmation"

var usage = fmt.Sprintf(`
Usage: nitro stop <container slug>

  stop a container

Options:

  -y, --yes=false    %s
`, yesUsage)

type Command struct {
	UI     command.UI
	Server string
	Client *httpclient.Client
}

func init() {
	command.RegisterAuth("stop", NewCommand)
}

func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI:     command.DefaultUI,
		Server: command.Env.APIURL,
		Client: httpclient.New(token),
	}, nil
}

func (c *Command) Run(args []string) int {
	var assumeYes bool

	cmdFlags := flag.NewFlagSet("stop", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.BoolVarP(&assumeYes, "yes", "y", false, yesUsage)

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) < 1 {
		c.UI.Output(c.Help())
		return 1
	}

	containerStr := args[0]

	if !assumeYes {
		confirmation := c.UI.Confirm(fmt.Sprintf("Do you really want to stop '%s'? [y/N]", containerStr))
		switch confirmation {
		case command.ConfirmYes:
			// Pass.
		case command.ConfirmNo:
			c.UI.Output("Cancelled.")
			return 1
		case command.ConfirmCancel:
			c.UI.Error("Cancelled.")
			return 130
		case command.ConfirmErr:
			c.UI.Error("An unexpected error has occurred.")
			return 1
		}
	}

	c.UI.Output(fmt.Sprintf("Stopping %s...", containerStr))
	container, err := c.stop(containerStr)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	t := PrettyPrint([]string{"Slug", "Image", "State"}, container)
	c.UI.Output(t)
	return 0
}

func (c *Command) Help() string {
	return strings.TrimSpace(usage)
}

func (c *Command) Synopsis() string {
	return "Stop a container"
}

func (c *Command) stop(container string) (*Container, error) {
	url := fmt.Sprintf("%s/api/v0/containers/%s/stop", c.Server, container)
	resp, err := c.Client.PostFormAuthed(url, nil)
	if err != nil {
		return nil, err
	}
	cnr, err := ParseContainerResponse(resp)
	if err != nil {
		return nil, err
	}
	return cnr, nil
}
