package login

import (
	"fmt"

	"strings"

	"github.com/bgentry/speakeasy"
	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/auth"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/config"
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
	UI     cli.Ui
	Server string
}

func init() {
	command.Register("login", NewLoginCommand)
}

func NewLoginCommand() (cli.Command, error) {
	return &LoginCommand{
		UI:     command.DefaultUI,
		Server: config.APIServerURL(),
	}, nil
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
			if err.Error() == "interrupted" {
				c.UI.Error("Login cancelled.")
				return 130
			}
			c.UI.Error(fmt.Sprintf("An unexpected error has occurred: %v", err))
		}
	}
	if password == "" {
		password, err = speakeasy.Ask("Password (will not be echoed): ")
		if err != nil {
			c.UI.Error(fmt.Sprintf("An unexpected error has occurred: %v", err))
		}
	}

	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	accessToken, err := auth.FetchToken(c.Server, username, password)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to login: %v", err))
		return 1
	}

	err = auth.SaveToken(c.Server, username, accessToken)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Error occurred when saving the access token: %v", err))
		return 1
	}

	c.UI.Info(fmt.Sprintf("Logged in as '%v'", username))
	return 0
}

func (c *LoginCommand) Synopsis() string {
	return "Login to Nitrous.IO"
}
