package login

import (
	"errors"
	"fmt"

	"strings"

	"github.com/bgentry/speakeasy"
	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/auth"
	"github.com/nitrous-io/nitro/command"
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

type Command struct {
	UI     cli.Ui
	Server string
}

func init() {
	command.Register("login", NewCommand)
}

func NewCommand() (cli.Command, error) {
	return &Command{
		UI:     command.DefaultUI,
		Server: command.Env.APIURL,
	}, nil
}

func (c *Command) Help() string {
	return strings.TrimSpace(fmt.Sprintf(usage, usernameUsage, passwordUsage))
}

func (c *Command) Run(args []string) int {
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

	if username == "" && password == "" {
		c.UI.Info("Login to Nitrous.IO:")
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

	c.UI.Info(fmt.Sprintf("Logged in as '%v'.", username))
	return 0
}

func (c *Command) Synopsis() string {
	return "Login to Nitrous.IO"
}

func PromptLogin() error {
	c, err := NewCommand()
	if err != nil {
		return err
	}

	s := c.Run([]string{})
	if s == 0 {
		// reload config
		command.Env = command.ConfigureEnvironment()
		return nil
	}
	return errors.New("Login failed")
}
