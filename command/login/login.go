package login

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/utils"
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
	ClientID      string
	ClientSecret  string
	Server        string
}

func init() {
	command.Register("login", NewLoginCommand)
}

func NewLoginCommand() (cli.Command, error) {
	config, err := utils.LoadConfig()
	if err != nil {
		return nil, errors.New("failed to load config")
	}
	return &LoginCommand{
		UI:            command.DefaultUI,
		TokenFilePath: utils.TokenFilePath(),
		ClientID:      config.ClientID,
		ClientSecret:  config.ClientSecret,
		Server:        config.Server,
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
				return 0
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

	config, err := utils.LoadConfig()
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to load config. Place your config file in the given path.\n: %v", err))
		return 1
	}

	p := url.Values{}
	p.Set("grant_type", "password")
	p.Set("username", username)
	p.Set("password", password)
	p.Set("client_id", config.ClientID)         // fetch from config
	p.Set("client_secret", config.ClientSecret) // fetch from config

	no_ssl_verify, _ := strconv.ParseBool(os.Getenv("NO_SSL_VERIFY"))
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: no_ssl_verify},
	}

	client := &http.Client{Transport: tr}
	url := fmt.Sprintf("%s/oauth/token", config.Server)
	response, err := client.PostForm(url, p)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to authenticate: %v", err))
	}
	defer response.Body.Close()

	type message struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		AccessToken      string `json:"access_token"`
	}
	var m message

	err = json.NewDecoder(response.Body).Decode(&m)
	if err != nil {
		c.UI.Error(fmt.Sprintf("An unexpected error has occurred: %v", err))
	}

	switch response.StatusCode {
	case 401:
		c.UI.Error(fmt.Sprintf("Failed to login: %v", m.ErrorDescription))
	case 200:
		err = ioutil.WriteFile(filepath.Join(os.Getenv("HOME"), ".nitro_token"), []byte(m.AccessToken), 0644)
		if err != nil {
			c.UI.Error(fmt.Sprintf("Error occurred when saving access token: %v", err))
		} else {
			c.UI.Info(fmt.Sprintf("Logged in as '%v'", username))
		}
	default:
		c.UI.Error(fmt.Sprintf("Failed to authenticate. Authentication responded with : %v", response.Status))
	}

	return 0
}

func (c *LoginCommand) Synopsis() string {
	return "Login to Nitrous.IO"
}
