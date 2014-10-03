package push

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/images"
	"github.com/nitrous-io/nitrous-common/httpclient"
	flag "github.com/ogier/pflag"
)

const usage = `
Usage: nitro images:push <host> <image id or repo url tag> [registry_host/]username/<repo_name:tag>

  Push image to a registry. If registry host is not specified, defaults to Docker Hub registry.
`

type Command struct {
	UI     *cli.BasicUi
	Server string
	Client *httpclient.Client
}

func init() {
	command.RegisterAuth("images:push", NewCommand)
}

func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI: &cli.BasicUi{
			Reader: os.Stdin,
			Writer: os.Stdout,
		},
		Server: command.Env.APIURL,
		Client: httpclient.New(token),
	}, nil
}

func (c *Command) Help() string {
	return strings.TrimSpace(usage)
}

func (c *Command) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("images:push", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 3 {
		c.UI.Output(c.Help())
		return 1
	}

	stdout, ok := c.UI.Writer.(*os.File)
	if !ok {
		c.UI.Error("Requires a working terminal")
		return 1
	}

	host, image, repo := args[0], args[1], args[2]
	c.UI.Output(fmt.Sprintf("Pushing image '%s' on '%s' to '%s'...", image, host, repo))

	if err := c.makeRequest(host, image, repo, stdout, stdout.Fd()); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.UI.Output("Pushed image")
	return 0
}

func (c *Command) Synopsis() string {
	return "Push image on a host to a registry"
}

func (c *Command) makeRequest(host, image, repo string, out io.Writer, termFd uintptr) error {
	path := fmt.Sprintf("%s/api/v0/hosts/%s/images/%s/push", c.Server, host, image)
	params := &url.Values{"repository": []string{repo}}
	resp, err := c.Client.PostFormAuthed(path, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("Failed: Login to the registry you want to push to first. See `nitro registry:login`.")
	}

	if err := httpclient.CheckResponse(resp, http.StatusOK); err != nil {
		return err
	}

	return images.DisplayJSONMessagesStream(resp.Body, out, termFd, true)
}
