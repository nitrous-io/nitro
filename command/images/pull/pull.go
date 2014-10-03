package pull

import (
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
Usage: nitro images:pull <host> [registry_host/][username/]<repo_name:tag>

  pull image onto a given host
`

type Response struct {
	Image images.Image `json:"image"`
}

type Command struct {
	UI     *cli.BasicUi
	Server string
	Client *httpclient.Client
}

func init() {
	command.RegisterAuth("images:pull", NewCommand)
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
	cmdFlags := flag.NewFlagSet("images:pull", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 2 {
		c.UI.Output(c.Help())
		return 1
	}

	stdout, ok := c.UI.Writer.(*os.File)
	if !ok {
		c.UI.Error("Requires a working terminal")
		return 1
	}

	host, image := args[0], args[1]
	c.UI.Output(fmt.Sprintf("Pulling image '%s' to '%s'...", image, host))
	if err := c.pullImage(host, image, stdout, stdout.Fd()); err != nil {
		c.UI.Error(fmt.Sprintf("images:pull %s %s failed. Error: %s", host, image, err))
		return 1
	}
	return 0
}

func (c *Command) Synopsis() string {
	return "Pull image onto a host"
}

func (c *Command) pullImage(host, image string, out io.Writer, termFd uintptr) error {
	path := fmt.Sprintf("%s/api/v0/hosts/%s/images/pull", c.Server, host)
	params := &url.Values{}
	params.Set("repository", image)
	params.Set("update", "true")
	resp, err := c.Client.PostFormAuthed(path, params)
	if err != nil {
		return fmt.Errorf("failed to pull image: %s", err)
	}
	defer resp.Body.Close()

	if err := httpclient.CheckResponse(resp, http.StatusOK); err != nil {
		return fmt.Errorf("unexpected response: %s", err)
	}

	if err := images.DisplayJSONMessagesStream(resp.Body, out, termFd, true); err != nil {
		return fmt.Errorf("failed to display download progress: %s", err)
	}
	return nil
}
