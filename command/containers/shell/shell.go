package shell

import (
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/pkg/httpshell"
	"github.com/nitrous-io/nitro/pkg/nitrousapi"
	"github.com/nitrous-io/nitrous-common/httpclient"
	flag "github.com/ogier/pflag"
)

const (
	usernameUsage = "Run as this user"
	defaultTerm   = "xterm-256color"
)

var usage = fmt.Sprintf(`
Usage: nitro shell <slug>

  open a shell on a container

Options:

  -u, --username=""    %s.
`, usernameUsage)

func init() {
	command.RegisterAuth("shell", NewCommand)
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

	cr, err := nitrousapi.GetContainer(c.client, c.server, slug)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	// prepare the term
	stdin := c.UI.Reader
	stdout := c.UI.Writer
	pty, ok := stdout.(*os.File)
	if !ok {
		c.UI.Error("Requires a working terminal")
		return 1
	}

	host := fmt.Sprintf("%s:%d", cr.Host.IPAddress, cr.Host.FreighterdPort)
	command := getLoginCommand(username)

	tlsConfig, err := nitrousapi.TLSConfig(cr.Host)
	if err != nil {
		c.UI.Error(fmt.Sprintf("failed to get secure HTTPS config %s", err))
		return 1
	}

	// start the shell
	err = httpshell.Run(host, cr.CID, command, tlsConfig, pty.Fd(), stdin, stdout)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Exited: %v", err))
		return 1
	}
	return 0
}

func (c *Command) Help() string {
	return strings.TrimSpace(fmt.Sprintf(usage))
}

func (c *Command) Synopsis() string {
	return "Open a shell on a container"
}

func getLoginCommand(username string) string {
	if username == "" {
		username = "root"
	}
	return fmt.Sprintf("sudo login -f %s", username)
}
