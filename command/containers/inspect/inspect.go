package inspect

import (
	"flag"
	"fmt"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/nitrous-io/nitro/command/containers"
	"github.com/nitrous-io/nitro/command/login"
)

const usage = `
Usage: nitro inspect <container slug>

  inspect a container
`

type Command struct {
	UI     cli.Ui
	Server string
	Client *httpclient.Client
}

func init() {
	command.Register("inspect", NewCommand)
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

	cmdFlags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	if len(args) != 1 {
		c.UI.Output(c.Help())
		return 1
	}

	containerStr := args[0]

	c.UI.Output(fmt.Sprintf("%s:", containerStr))
	container, err := c.inspectContainer(containerStr)
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
	return "Inspect a container"
}

func (c *Command) inspectContainer(container string) (*Container, error) {
	url := fmt.Sprintf("%s/api/v0/containers/%s", c.Server, container)
	resp, err := c.Client.GetAuthed(url, nil)
	if err != nil {
		return nil, err
	}
	cnr, err := ParseContainerResponse(resp)
	if err != nil {
		return nil, err
	}
	return cnr, nil
}
