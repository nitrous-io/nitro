package reconfig

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	flag "github.com/ogier/pflag"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitrous-common/httpclient"

	"github.com/nitrous-io/nitro/command/containers"
)

const usage = `
Usage: nitro reconfig [options] <container slug>

  reconfigure a container

Options:

  -e, --env=[]            %s
  -v, --volume=[]         %s
  -p, --port-forward=[]   %s
  -P, --port-forward-all  %s
  -c, --command=""        %s
`

type Command struct {
	UI     cli.Ui
	Server string
	Client *httpclient.Client
}

type containerParams struct {
	envVars        command.Multiarg
	volumes        command.Multiarg
	portForwards   command.Multiarg
	portForwardAll bool
	cmd            string
}

func init() {
	command.RegisterAuth("reconfig", NewCommand)
}

func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI:     command.DefaultUI,
		Server: command.Env.APIURL,
		Client: httpclient.New(token),
	}, nil
}

func (c *Command) Help() string {
	return strings.TrimSpace(fmt.Sprintf(usage,
		containers.EnvUsage,
		containers.VolumesUsage,
		containers.PortForwardsUsage,
		containers.PortForwardAllUsage,
		containers.Cmd))
}

func (c *Command) Run(args []string) int {
	params := &containerParams{}

	cmdFlags := flag.NewFlagSet("reconfig", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	cmdFlags.VarP(&params.envVars, "env", "e", containers.EnvUsage)
	cmdFlags.VarP(&params.volumes, "volume", "v", containers.VolumesUsage)
	cmdFlags.VarP(&params.portForwards, "port-forward", "p", containers.PortForwardsUsage)
	cmdFlags.BoolVarP(&params.portForwardAll, "port-forward-all", "P", false, containers.PortForwardAllUsage)
	cmdFlags.StringVarP(&params.cmd, "command", "c", "", containers.Cmd)

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		c.UI.Output(c.Help())
		return 1
	}
	slug := args[0]

	c.UI.Output(fmt.Sprintf("Reconfiguring '%s'. Please wait...", slug))
	container, err := c.reconfig(slug, params)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to reconfigure the container - %v", err.Error()))
		return 1
	}

	t := containers.PrettyPrint([]string{"Slug", "Image", "State"}, container)
	c.UI.Output(t)
	return 0
}

func (c *Command) Synopsis() string {
	return "Reconfigure a container"
}

func (c *Command) reconfig(slug string, cp *containerParams) (*containers.Container, error) {
	p := &url.Values{}

	for _, e := range cp.envVars {
		p.Add("env[]", e)
	}
	for _, v := range cp.volumes {
		p.Add("volume[]", v)
	}
	for _, pf := range cp.portForwards {
		p.Add("port[]", pf)
	}

	if cp.cmd != "" {
		p.Add("cmd", cp.cmd)
	}

	resp, err := c.Client.PostFormAuthed(fmt.Sprintf("%s/api/v0/containers/%s/reconfig", c.Server, slug), p)
	if err != nil {
		return nil, err
	}
	err = httpclient.CheckResponse(resp, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	j := struct {
		Container *containers.Container `json:"container"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&j)
	if err != nil {
		return nil, err
	}

	return j.Container, nil
}
