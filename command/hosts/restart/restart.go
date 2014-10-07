package restart

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitrous-common/httpclient"
	flag "github.com/ogier/pflag"
)

const (
	yesUsage   = "Skip confirmation"
	forceUsage = "Restart host even if it is not running"
)

var usage = fmt.Sprintf(`
Usage: nitro restart <slug>

  restart a host

Options:

  -f, --force=false    %s
  -y, --yes=false      %s
`, yesUsage, forceUsage)

type Command struct {
	UI     command.UI
	Server string
	Client *httpclient.Client
}

func init() {
	command.RegisterAuth("hosts:restart", NewCommand)
}

func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI:     command.DefaultUI,
		Server: command.Env.APIURL,
		Client: httpclient.New(token),
	}, nil
}

func (c *Command) Run(args []string) int {
	var force, assumeYes bool

	cmdFlags := flag.NewFlagSet("hosts:restart", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.BoolVarP(&force, "force", "f", false, forceUsage)
	cmdFlags.BoolVarP(&assumeYes, "yes", "y", false, yesUsage)

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) < 1 {
		c.UI.Output(c.Help())
		return 1
	}

	slug := args[0]

	if !assumeYes {
		confirmation := c.UI.Confirm(fmt.Sprintf("Do you really want to restart '%s'? [y/N]", slug))
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

	c.UI.Output(fmt.Sprintf("Restarting host %s...", slug))
	err = c.restart(slug, force)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed: %s", err.Error()))
		return 1
	}

	c.UI.Output("Done.")
	return 0
}

func (c *Command) Help() string {
	return strings.TrimSpace(usage)
}

func (c *Command) Synopsis() string {
	return "Restart a host"
}

func (c *Command) restart(slug string, force bool) error {
	u := fmt.Sprintf("%s/api/v0/hosts/%s/restart", c.Server, slug)
	params := &url.Values{}
	if force {
		params.Set("force", "true")
	}
	resp, err := c.Client.PostFormAuthed(u, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return httpclient.CheckResponse(resp, http.StatusOK)
}
