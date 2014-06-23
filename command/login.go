package command

import (
	"fmt"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/environment"
	flag "github.com/ogier/pflag"
)

const (
	usernameUsage = "Nitrous.IO username"
	passwordUsage = "Nitrous.IO password"
)

const usage = `
Usage: nitro login [options]

  login authenticates with your Nitrous.IO account.

Options:

  -u, --user=""    %s.
  -p, --pass=""    %s.
`

type LoginCommand struct {
	UI  cli.Ui
	Env environment.Environment
}

func (c *LoginCommand) Help() string {
	return strings.TrimSpace(fmt.Sprintf(usage, usernameUsage, passwordUsage))
}

func (c *LoginCommand) Run(args []string) int {
	var username string
	var password string

	cmdFlags := flag.NewFlagSet("login", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.StringVarP(&username, "user", "u", "", usernameUsage)
	cmdFlags.StringVarP(&password, "pass", "p", "", passwordUsage)

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	if username == "" {
		username, err = c.UI.Ask("Username:")
		if err != nil {
			c.UI.Error(fmt.Sprintf("An unexpected error has occurred: %v", err))
		}
	}
	if password == "" {
		c.UI.Output("Password (will not be echoed): ")
		p := gopass.GetPasswd()
		password = string(p)
	}
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	c.UI.Info(fmt.Sprintf("Logging in with '%v', '%v'", username, password))
	// TODO Actually log in.
	c.UI.Info(fmt.Sprintf("Logged in as '%v'", username))

	return 0
}

func (c *LoginCommand) Synopsis() string {
	return "Login to Nitrous.IO"
}
