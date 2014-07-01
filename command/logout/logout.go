package logout

import (
	"fmt"

	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/auth"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/config"
	flag "github.com/ogier/pflag"
)

const usage = `
Usage: nitro logout

  logout from authenticated Nitrous.IO account.
`

type LogoutCommand struct {
	UI     cli.Ui
	Server string
}

func init() {
	command.Register("logout", NewLogoutCommand)
}

func NewLogoutCommand() (cli.Command, error) {
	return &LogoutCommand{
		UI:     command.DefaultUI,
		Server: config.APIServer(),
	}, nil
}

func (c *LogoutCommand) Help() string {
	return strings.TrimSpace(fmt.Sprintf(usage))
}

func (c *LogoutCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("logout", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	err = auth.RemoveToken(c.Server)
	if err != nil {
		c.UI.Error("Failed to remove the token.")
		return 1
	}

	c.UI.Info("You are no longer logged-in to Nitrous.IO")
	return 0
}

func (c *LogoutCommand) Synopsis() string {
	return "Logout from authenticated Nitrous.IO account"
}
