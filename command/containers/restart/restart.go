package restart

import (
	"fmt"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitrous-common/httpclient"
	flag "github.com/ogier/pflag"

	. "github.com/nitrous-io/nitro/command/containers"
	"github.com/nitrous-io/nitro/command/login"
)

const forceUsage = "Skip confirmation"

const usage = `
Usage: nitro restart <container slug>

  restart a container

Options:

  -f, --force=""    %s
`

type Command struct {
	UI     command.UI
	Server string
	Client *httpclient.Client
}

func init() {
	command.Register("restart", NewCommand)
}

func NewCommand() (cli.Command, error) {
	return &Command{
		UI:     command.DefaultUI,
		Server: command.Env.APIURL,
		Client: httpclient.New(command.Env.AuthToken),
	}, nil
}

func (c *Command) Run(args []string) int {
	if command.Env.AuthToken == "" {
		err := login.PromptLogin()
		if err != nil {
			return 1
		}
		c.Client = httpclient.New(command.Env.AuthToken)
	}

	var force bool

	cmdFlags := flag.NewFlagSet("restart", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.BoolVarP(&force, "force", "f", false, forceUsage)

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	if len(args) < 1 {
		c.UI.Output(c.Help())
		return 1
	}

	containerStr := args[0]

	if !force {
		confirmation := c.UI.Confirm(fmt.Sprintf("Do you really want to restart '%s'? [y/N]", containerStr))
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
			c.UI.Error(fmt.Sprintf("An unexpected error has occurred: %v", err))
			return 1
		}
	}

	c.UI.Output(fmt.Sprintf("Restarting %s...", containerStr))
	container, err := c.restart(containerStr)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	t := PrettyPrint([]string{"Slug", "Image", "State"}, container)
	c.UI.Output(t)
	return 0
}

func (c *Command) Help() string {
	return strings.TrimSpace(fmt.Sprintf(usage, forceUsage))
}

func (c *Command) Synopsis() string {
	return "Restart a container"
}

func (c *Command) restart(container string) (*Container, error) {
	url := fmt.Sprintf("%s/api/v0/containers/%s/restart", c.Server, container)
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
