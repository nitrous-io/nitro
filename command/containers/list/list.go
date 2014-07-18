package list

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/nitrous-io/nitro/command/containers"
	"github.com/nitrous-io/nitro/command/login"
)

const usage = `
Usage: nitro list <host>

  list containers in the host
`

type Command struct {
	UI     cli.Ui
	Server string
	Client *httpclient.Client
}

func init() {
	command.Register("list", NewCommand)
}

func NewCommand() (cli.Command, error) {
	return &Command{
		UI:     command.DefaultUI,
		Server: command.Env.APIURL,
		Client: httpclient.New(command.Env.AuthToken),
	}, nil
}

func (c *Command) Help() string {
	return strings.TrimSpace(usage)
}

func (c *Command) Run(args []string) int {
	if command.Env.AuthToken == "" {
		err := login.PromptLogin()
		if err != nil {
			return 1
		}
		c.Client = httpclient.New(command.Env.AuthToken)
	}

	cmdFlags := flag.NewFlagSet("list", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	var host string
	args = cmdFlags.Args()
	switch len(args) {
	case 0:
		host = command.DefaultHost()
		if host == "" {
			c.UI.Output(c.Help())
			return 1
		}
	case 1:
		host = args[0]
	default:
		c.UI.Output(c.Help())
		return 1
	}

	c.UI.Output(fmt.Sprintf("Listing containers on '%s'...", host))
	containers, err := c.fetchContainers(host)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}
	command.SaveDefaultHost(host)

	if len(containers) == 0 {
		c.UI.Output("No containers found.")
		return 0
	}

	t := PrettyPrint([]string{"Name", "Image", "State"}, containers...)
	c.UI.Output(t)

	return 0
}

func (c *Command) Synopsis() string {
	return "List containers"
}

func (c *Command) fetchContainers(host string) ([]*Container, error) {
	resp, err := c.Client.GetAuthed(fmt.Sprintf("%s/api/v0/containers?host=%s", c.Server, host), nil)
	if err != nil {
		return nil, err
	}
	err = httpclient.CheckResponse(resp, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	j := struct {
		Containers []*Container `json:"containers"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&j)
	if err != nil {
		return nil, err
	}

	return j.Containers, nil
}
