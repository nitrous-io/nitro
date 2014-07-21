package destroy

import (
	"fmt"
	"net/http"
	"strings"

	flag "github.com/ogier/pflag"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitrous-common/httpclient"
)

const forceUsage = "Skip confirmation"

const usage = `
Usage: nitro hosts:destroy <slug>

  deletes the host identified by the slug

Options:

  -f, --force=""    %s
`

type Command struct {
	UI     command.UI
	Server string
	Client *httpclient.Client
}

func init() {
	command.Register("hosts:destroy", NewCommand)
}

func NewCommand() (cli.Command, error) {
	return &Command{
		UI:     command.DefaultUI,
		Server: command.Env.APIURL,
		Client: httpclient.New(command.Env.AuthToken),
	}, nil
}

func (c *Command) Help() string {
	return strings.TrimSpace(usage)
}

func (c *Command) Run(args []string) int {
	var force bool

	cmdFlags := flag.NewFlagSet("hosts:destroy", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.BoolVarP(&force, "force", "f", false, forceUsage)

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		c.UI.Output(c.Help())
		return 1
	}

	name := args[0]

	if !force {
		confirmation := c.UI.Confirm(fmt.Sprintf("Do you really want to destroy '%s'? [y/N]", name))
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

	c.UI.Output(fmt.Sprintf("Deleting host '%s'. Please wait...", name))
	err = c.destroyHost(name)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to delete host - %v", err.Error()))
		return 1
	}

	// If we just deleted the default host, unset it.
	if command.DefaultHost() == name {
		command.SaveDefaultHost("")
	}

	c.UI.Output("Done.")
	return 0
}

func (c *Command) Synopsis() string {
	return "Deletes the given host"
}

func (c *Command) destroyHost(name string) error {
	resp, err := c.Client.DeleteAuthed(fmt.Sprintf("%s/api/v0/%s/%s", c.Server, "hosts", name), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return httpclient.CheckResponse(resp, http.StatusOK)
}
