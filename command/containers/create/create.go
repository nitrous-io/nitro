package create

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	flag "github.com/ogier/pflag"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/pkg/randomnamegenerator"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/nitrous-io/nitro/command/containers"
)

var usage = command.ContainerCreateUsage

type Command struct {
	UI     cli.Ui
	Server string
	Client *httpclient.Client
}

type containerParams struct {
	host         string
	slug         string
	image        string
	envVars      command.Multiarg
	volumes      command.Multiarg
	portForwards command.Multiarg
}

func init() {
	command.RegisterAuth("create", NewCommand)
}

func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI:     command.DefaultUI,
		Server: command.Env.APIURL,
		Client: httpclient.New(token),
	}, nil
}

func (c *Command) Help() string {
	return usage.String()
}

func (c *Command) Run(args []string) int {
	params := &containerParams{}

	cmdFlags := flag.NewFlagSet("create", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.StringVarP(&params.slug, "slug", "s", "", usage.OptUsage("slug"))
	cmdFlags.StringVarP(&params.image, "image", "i", "", usage.OptUsage("image"))

	cmdFlags.VarP(&params.envVars, "env", "e", usage.OptUsage("env"))
	cmdFlags.VarP(&params.volumes, "volume", "v", usage.OptUsage("volume"))
	cmdFlags.VarP(&params.portForwards, "port-forward", "p", usage.OptUsage("port-forward"))

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		c.UI.Error(fmt.Sprintf("Please specify a host."))
		c.UI.Output(c.Help())
		return 1
	}
	params.host = args[0]

	if params.slug == "" {
		params.slug = randomnamegenerator.Generate("container")
	}

	c.UI.Output(fmt.Sprintf("Creating container on '%s'. Please wait...", params.host))
	cnr, err := c.createContainer(params)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to create the container - %v", err.Error()))
		return 1
	}

	if cnr != nil {
		c.UI.Output(fmt.Sprintf("Created new container: '%s'", cnr.Slug))
		t := PrettyPrint([]string{"Slug", "Image", "State"}, cnr)
		c.UI.Output(t)
	} else {
		c.UI.Output("Done.")
	}
	return 0
}

func (c *Command) Synopsis() string {
	return usage.Synopsis
}

func (c *Command) createContainer(cp *containerParams) (*Container, error) {
	p := &url.Values{}

	p.Set("host", cp.host)
	if cp.slug != "" {
		p.Set("slug", cp.slug)
	}
	if cp.image != "" {
		p.Set("image", cp.image)
	}
	for _, e := range cp.envVars {
		p.Add("env[]", e)
	}
	for _, v := range cp.volumes {
		p.Add("volume[]", v)
	}
	for _, pf := range cp.portForwards {
		p.Add("port[]", pf)
	}

	resp, err := c.Client.PostFormAuthed(fmt.Sprintf("%s/api/v0/containers", c.Server), p)
	if err != nil {
		return nil, err
	}
	err = httpclient.CheckResponse(resp, http.StatusCreated)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	j := struct {
		Container *Container `json:"container"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&j)
	if err != nil {
		return nil, err
	}

	return j.Container, nil
}
