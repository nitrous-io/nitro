package clear

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitrous-common/httpclient"
	flag "github.com/ogier/pflag"
)

const yesUsage = "Skip confirmation"

var usage = fmt.Sprintf(`
Usage: nitro keys:clear <container>

  remove all public keys from container

Options:

  -y, --yes=false    %s
`, yesUsage)

type Command struct {
	UI     command.UI
	Server string
	Client *httpclient.Client
}

func init() {
	command.RegisterAuth("keys:clear", NewCommand)
}

func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI:     command.DefaultUI,
		Server: command.Env.APIURL,
		Client: httpclient.New(token),
	}, nil
}

func (c *Command) Help() string {
	return strings.TrimSpace(usage)
}

func (c *Command) Run(args []string) int {
	var assumeYes bool
	cmdFlags := flag.NewFlagSet("keys:clear", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.BoolVarP(&assumeYes, "yes", "y", false, yesUsage)

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}
	args = cmdFlags.Args()
	if len(args) != 1 {
		c.UI.Output(c.Help())
		return 1
	}

	container := args[0]

	if !assumeYes {
		confirmation := c.UI.Confirm(fmt.Sprintf("Do you really want to clear all public keys in '%s'? [y/N]", container))
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

	err = c.makeRequest(container)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}
	c.UI.Output(fmt.Sprintf("All public keys removed for container %s", container))
	return 0
}

func (c *Command) makeRequest(container string) error {
	path := fmt.Sprintf("%s/api/v0/containers/%s/keys", c.Server, container)
	resp, err := c.Client.DeleteAuthed(path, nil)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	return command.CheckResponse(resp, http.StatusOK)
}

func (c *Command) Synopsis() string {
	return "Remove all public keys from container"
}
