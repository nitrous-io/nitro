package push

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
	Usage = `
Usage: nitro images:push <host> <image id or repo url tag> [registry_host/]username/<repo_name:tag>

  push image to a registry.  If registry host is not specified, defaults to hub.docker.com
`
)

type Command struct {
	UI     command.UI
	Server string
	Client *httpclient.Client
}

func init() {
	command.RegisterAuth("images:push", NewCommand)
}

func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI:     command.DefaultUI,
		Server: command.Env.APIURL,
		Client: httpclient.New(token),
	}, nil
}

func (c *Command) Help() string {
	return strings.TrimSpace(Usage)
}

func (c *Command) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("images:push", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 3 {
		c.UI.Output(c.Help())
		return 1
	}

	host, image, dest := args[0], args[1], args[2]

	c.UI.Output(fmt.Sprintf("Pushing image '%s' on '%s' to '%s'...", image, host, dest))

	if err := c.makeRequest(host, image, dest); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.UI.Output("Pushed image")
	return 0
}

func (c *Command) Synopsis() string {
	return "Push image on a host to a registry"
}

func (c *Command) makeRequest(host, image, dest string) error {
	path := fmt.Sprintf("%s/api/v0/hosts/%s/images/push", c.Server, host)
	params := &url.Values{}
	params.Set("image", image)
	params.Set("destination", dest)
	resp, err := c.Client.PostFormAuthed(path, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return httpclient.CheckResponse(resp, http.StatusOK)
}
