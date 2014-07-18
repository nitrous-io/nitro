package destroy

import (
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/login"
	"github.com/nitrous-io/nitrous-common/httpclient"
)

const usage = `
Usage: nitro hosts:destroy <slug>

  deletes the host identified by the slug
`

type Command struct {
	UI     cli.Ui
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
	if command.Env.AuthToken == "" {
		err := login.PromptLogin()
		if err != nil {
			return 1
		}
		c.Client = httpclient.New(command.Env.AuthToken)
	}

	cmdFlags := flag.NewFlagSet("hosts:destroy", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

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

	c.UI.Output("Deleting host. Please wait...")
	err = c.destroyHost(name)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to delete host - %v", err.Error()))
		return 1
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
