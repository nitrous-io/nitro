package login

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/utils"
	"github.com/nitrous-io/nitro/user"
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
	UI            cli.Ui
	TokenFilePath string
	Config        *utils.Config
}

func init() {
	command.Register("login", NewLoginCommand)
}

func NewLoginCommand() (cli.Command, error) {
	config, err := utils.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config file")
	}
	return &LoginCommand{
		UI:            command.DefaultUI,
		TokenFilePath: utils.TokenFilePath(),
		Config:        config,
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
				return 2
			}
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

	accessToken, err := user.GetAccessToken(username, password, c.Config)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to login: %v", err))
		return 1
	}

	err = ioutil.WriteFile(c.TokenFilePath, []byte(accessToken), 0644)
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
