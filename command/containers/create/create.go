package create

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	flag "github.com/ogier/pflag"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/pkg/randomnamegenerator"
	"github.com/nitrous-io/nitrous-common/httpclient"

	. "github.com/nitrous-io/nitro/command/containers"
	"github.com/nitrous-io/nitro/command/images"
)

const (
	slugUsage         = "A unique slug for the container"
	imageUsage        = "Specify an image to use to create the container"
	envUsage          = "Set environment variables for the container in the format <key>:<value>. " + command.MultiargUsage
	volumesUsage      = "Add data volume so other containers can mount given <path>. " + command.MultiargUsage
	portForwardsUsage = "Forward ports from container to host.  Use one of: 'ip:hostPort:containerPort', 'ip::containerPort', or 'hostPort:containerPort' specifications. " + command.MultiargUsage

	defaultImage = "nitrousio/ubuntu:trusty"
)

const usage = `
Usage: nitro create <host> [options]

  create a new container

Options:

  -s, --slug=""           %s
  -i, --image=""          %s
  -e, --env=[]            %s
  -v, --volume=[]         %s
  -p, --port-forward=[]   %s
`

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

type imageResponse struct {
	Image images.Image `json:"image"`
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
	// TODO this is getting unwieldy, maybe text/template
	return strings.TrimSpace(fmt.Sprintf(usage, slugUsage,
		imageUsage, envUsage, volumesUsage, portForwardsUsage))
}

func (c *Command) Run(args []string) int {
	params := &containerParams{}

	cmdFlags := flag.NewFlagSet("create", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.StringVarP(&params.slug, "slug", "s", "", slugUsage)
	cmdFlags.StringVarP(&params.image, "image", "i", "", imageUsage)

	cmdFlags.VarP(&params.envVars, "env", "e", envUsage)
	cmdFlags.VarP(&params.volumes, "volume", "v", volumesUsage)
	cmdFlags.VarP(&params.portForwards, "port-forward", "p", portForwardsUsage)

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

	if params.image == "" {
		c.UI.Output(fmt.Sprintf("Using default image %s", defaultImage))
		params.image = defaultImage
	}

	c.UI.Output(fmt.Sprintf("Pulling image '%s' on '%s'. Please wait...", params.image, params.host))
	if err := c.pullImage(params.host, params.image); err != nil {
		c.UI.Error(fmt.Sprintf("Failed to pull image %s on host %s: %s",
			params.image, params.host, err))
		return 1
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
	return "Create a new container"
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

func (c *Command) pullImage(host, image string) error {
	path := fmt.Sprintf("%s/api/v0/hosts/%s/images/pull", c.Server, host)
	params := &url.Values{"repository": []string{image}}
	resp, err := c.Client.PostFormAuthed(path, params)
	if err != nil {
		return err
	}
	if err := httpclient.CheckResponse(resp, http.StatusOK); err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
