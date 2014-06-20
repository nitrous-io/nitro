package command

import (
	"fmt"
	"os"

	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command/utils"
	flag "github.com/ogier/pflag"
)

const logoutUsage = `
Usage: nitro logout

  logout from authenticated Nitrous.IO account.
`

type LogoutCommand struct {
	UI cli.Ui
}

func (c *LogoutCommand) Help() string {
	return strings.TrimSpace(fmt.Sprintf(logoutUsage))
}

func (c *LogoutCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("logout", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	err = os.Remove(utils.TokenFilePath())
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to logout from Nitrous.IO"))
		return 1
	}

	c.UI.Info(fmt.Sprintf("Successfully logged out from Nitrous.IO"))
	return 0
}

func (c *LogoutCommand) Synopsis() string {
	return "Logout from authenticated Nitrous.IO account"
}
