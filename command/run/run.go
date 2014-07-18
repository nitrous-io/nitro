package run

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/pkg/httpshell"
	"github.com/nitrous-io/nitrous-common/httpclient"
)

const usage = `
Usage: nitro run [container] [cmd]

run a command on a remote docker container.
`

func init() {
	command.RegisterAuth("run", NewCommand)
}

type Command struct {
	UI     *cli.BasicUi
	client *httpclient.Client
	server string
}

func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI: &cli.BasicUi{
			Reader: os.Stdin,
			Writer: os.Stdout,
		},
		server: command.Env.APIURL,
		client: httpclient.New(token),
	}, nil
}

func (c *Command) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("run", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) < 2 {
		c.UI.Error(c.Help())
		return 1
	}

	slug := args[0]
	cr, err := c.getContainerResp(slug)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	host := fmt.Sprintf("%s:%d", cr.Host.IpAddress.String, cr.Host.FreighterdPort.Int64)
	command := strings.Join(args[1:], " ")

	stdin := c.UI.Reader
	stdout := c.UI.Writer
	pty, ok := stdout.(*os.File)
	if !ok {
		c.UI.Error("Requires a working terminal")
		return 1
	}

	err = httpshell.Run(host, cr.Container.CID, command, pty.Fd(), stdin, stdout)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Exited: %v", err))
		return 1
	}
	return 0
}

func (c *Command) Help() string {
	return strings.TrimSpace(usage)
}

func (c *Command) Synopsis() string {
	return "Run a command on a remote box"
}

type containerResp struct {
	Container struct {
		CID   string `json:"cid"`
		Slug  string `json:"slug"`
		Image string `json:"image"`
		State string `json:"state"`
	} `json:"container"`

	Host struct {
		Slug           string
		IpAddress      sql.NullString
		FreighterdPort sql.NullInt64
	} `json:"host"`
}

func (c *Command) getContainerResp(slug string) (*containerResp, error) {
	// get container data
	endpoint := fmt.Sprintf("%s/api/v0/containers/%s", c.server, slug)
	resp, err := c.client.GetAuthed(endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	cr := &containerResp{}
	if err := json.NewDecoder(resp.Body).Decode(cr); err != nil {
		return nil, err
	}
	return cr, nil
}
