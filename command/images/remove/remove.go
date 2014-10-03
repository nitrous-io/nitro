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
Usage: nitro images:remove <host> <image id or name[:tag]>

  remove image on a given host

Options:

  -y, --yes=false    %s
`, yesUsage)

type Command struct {
	UI     command.UI
	Server string
	Client *httpclient.Client
}

func init() {
	command.RegisterAuth("images:remove", NewCommand)
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
	cmdFlags := flag.NewFlagSet("images:remove", flag.ContinueOnError)
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

	host, image := args[0], args[1]

	if !assumeYes {
		confirmation := c.UI.Confirm(fmt.Sprintf("Do you really want to remove image '%s' from '%s'? [y/N]", image, host))
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

	c.UI.Output(fmt.Sprintf("Removing image '%s' on '%s'...", image, host))

	if err := c.makeRequest(host, image); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.UI.Output(fmt.Sprintf("Removed image '%s' from host '%s'", image, host))
	return 0
}

func (c *Command) Synopsis() string {
	return "Remove image on a host"
}

func (c *Command) makeRequest(host, image string) error {
	path := fmt.Sprintf("%s/api/v0/hosts/%s/images/%s", c.Server, host, image)
	resp, err := c.Client.DeleteAuthed(path, nil)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	return httpclient.CheckResponse(resp, http.StatusOK)
}
