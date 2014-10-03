package remove

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
Usage: nitro keys:remove <container> <md5_fingerprint>

  remove a public key

Options:

  -y, --yes=false    %s
`, yesUsage)

type Command struct {
	UI     command.UI
	Server string
	Client *httpclient.Client
}

func init() {
	command.RegisterAuth("keys:remove", RemoveCommand)
}

func RemoveCommand(token string) (cli.Command, error) {
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
	cmdFlags := flag.NewFlagSet("keys:remove", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.BoolVarP(&assumeYes, "yes", "y", false, yesUsage)

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}
	args = cmdFlags.Args()
	if len(args) != 2 {
		c.UI.Output(c.Help())
		return 1
	}

	container := args[0]
	fingerprint := args[1]

	if !assumeYes {
		confirmation := c.UI.Confirm(fmt.Sprintf("Do you really want to remove this key from '%s'? [y/N]", container))
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

	err = c.makeRequest(container, fingerprint)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.UI.Output(fmt.Sprintf("Removed key %s from container %s", fingerprint, container))
	return 0
}

func (c *Command) makeRequest(container, fingerprint string) error {
	path := fmt.Sprintf("%s/api/v0/containers/%s/keys/%s", c.Server, container, fingerprint)
	resp, err := c.Client.DeleteAuthed(path, nil)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	return httpclient.CheckResponse(resp, http.StatusOK)
}

func (c *Command) Synopsis() string {
	return "Remove public key from container"
}
