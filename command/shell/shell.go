package shell

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/pkg/httpshell"
	"github.com/nitrous-io/nitrous-common/httpclient"
	flag "github.com/ogier/pflag"
)

const (
	usernameUsage = "Run as this user"
	defaultTerm   = "xterm-256color"
	synopsys      = "Open a shell on Nitrous.IO box"
)

var usage = fmt.Sprintf(`
Usage: nitro shell <slug>

open a shell to a box on a Nitrous.IO zone

Options:

  -u, --username=""    %s.
`, usernameUsage)

func init() {
	command.RegisterAuth("shell", NewCommand)
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

type Command struct {
	UI     *cli.BasicUi
	client *httpclient.Client
	server string
}

func (c *Command) Help() string {
	return strings.TrimSpace(fmt.Sprintf(usage))
}

func (c *Command) Synopsis() string {
	return synopsys
}

func (c *Command) Run(args []string) int {
	var username string
	cmdFlags := flag.NewFlagSet("shell", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.StringVarP(&username, "username", "u", "", usernameUsage)

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
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

	command := getLoginCommand(username)

	// prepare the term
	stdin := c.UI.Reader
	stdout := c.UI.Writer
	pty, ok := stdout.(*os.File)
	if !ok {
		c.UI.Error("nitro shell requires a working terminal")
		return 1
	}

	err = httpshell.Run(host, cr.Container.CID, command, pty.Fd(), stdin, stdout)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Exited: %v", err))
		return 1
	}
	return 0
}

func getLoginCommand(username string) string {
	if username == "" {
		username = "root"
	}
	return fmt.Sprintf("sudo login -f %s", username)
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
