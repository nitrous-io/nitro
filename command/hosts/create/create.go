package create

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/login"
	"github.com/nitrous-io/nitro/pkg/randomnamegenerator"
	"github.com/nitrous-io/nitrous-common/httpclient"
	flag "github.com/ogier/pflag"
)

const (
	regionUsage = "Region in which you want to create the host"
	slugUsage   = "A unique slug for the host"
)

const usage = `
Usage: nitro hosts:create <size>

  create a new host

Options:

  size
  -r, --region=""       %s
  -s, --slug=""         %s
`

type Command struct {
	UI     cli.Ui
	Server string
	Client *httpclient.Client
}

func init() {
	command.Register("hosts:create", NewCommand)
}

func NewCommand() (cli.Command, error) {
	return &Command{
		UI:     command.DefaultUI,
		Server: command.Env.APIURL,
		Client: httpclient.New(command.Env.AuthToken),
	}, nil
}

func (c *Command) Help() string {
	return strings.TrimSpace(fmt.Sprintf(usage, regionUsage, slugUsage))
}

func (c *Command) Run(args []string) int {
	if command.Env.AuthToken == "" {
		err := login.PromptLogin()
		if err != nil {
			return 1
		}
		c.Client = httpclient.New(command.Env.AuthToken)
	}

	var (
		region string
		slug   string
	)
	cmdFlags := flag.NewFlagSet("hosts:create", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.StringVarP(&region, "region", "r", "", regionUsage)
	cmdFlags.StringVarP(&slug, "slug", "s", "", slugUsage)

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}
	args = cmdFlags.Args()
	if len(args) != 1 {
		c.UI.Output(c.Help())
		return 1
	}

	if slug == "" {
		slug = randomnamegenerator.Generate("host")
	}

	size := args[0]

	c.UI.Output("Creating the host. Please wait...")
	err = c.createHost(size, region, slug)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to create the host - %v", err.Error()))
		return 1
	}

	c.UI.Output("Done.")
	return 0
}

func (c *Command) Synopsis() string {
	return "Creates a new host"
}

func (c *Command) createHost(size, region, slug string) error {
	p := &url.Values{}
	if size != "" {
		p.Set("size", size)
	}

	if slug != "" {
		p.Set("slug", slug)
	}

	if region != "" {
		p.Set("region", region)
	}

	resp, err := c.Client.PostFormAuthed(fmt.Sprintf("%s/api/v0/%s", c.Server, "hosts"), p)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return httpclient.CheckResponse(resp, http.StatusCreated)
}
