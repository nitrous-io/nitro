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
Usage: nitro volumes:remove <host> <volume_slug>

  remove a data volume from a host

Options:

  -y, --yes=false    %s
`, yesUsage)

type Command struct {
	UI     command.UI
	Server string
	Client *httpclient.Client
}

func init() {
	command.RegisterAuth("volumes:remove", RemoveCommand)
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
	cmdFlags := flag.NewFlagSet("volumes:remove", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.BoolVarP(&assumeYes, "yes", "y", false, yesUsage)

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}
	args = cmdFlags.Args()
	if len(args) != 2 {
		c.UI.Output(c.Help())
		return 1
	}

	hostSlug := args[0]
	volSlug := args[1]

	if !assumeYes {
		c.UI.Output("WARNING: All files in data volume will be removed!")
		confirmation := c.UI.Confirm(fmt.Sprintf("Do you really want to remove volume '%s' from '%s'? [y/N]", volSlug, hostSlug))
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

	if err := c.makeRequest(hostSlug, volSlug); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.UI.Output(fmt.Sprintf("Removed data volume %s from host %s", volSlug, hostSlug))
	return 0
}

func (c *Command) makeRequest(hostSlug, volSlug string) error {
	path := fmt.Sprintf("%s/api/v0/hosts/%s/volumes/%s", c.Server, hostSlug, volSlug)
	resp, err := c.Client.DeleteAuthed(path, nil)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	return httpclient.CheckResponse(resp, http.StatusOK)
}

func (c *Command) Synopsis() string {
	return "Remove data volume from a host"
}
