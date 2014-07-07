package shell

import (
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/pkg/httpshell"
	flag "github.com/ogier/pflag"
)

const (
	usernameUsage = "Run as this user"
	defaultTerm   = "xterm-256color"
	synopsys      = "Open a shell on Nitrous.IO box"
)

var usage = fmt.Sprintf(`
Usage: nitro shell <host_name> <container_id>

open a shell to a box on a Nitrous.IO zone

Options:

  -u, --username=""    %s.
`, usernameUsage)

func init() {
	command.Register("shell", NewShellCommand)
}

func NewShellCommand() (cli.Command, error) {
	return &ShellCommand{
		UI: &cli.BasicUi{
			Reader: os.Stdin,
			Writer: os.Stdout,
		},
	}, nil
}

type ShellCommand struct {
	UI *cli.BasicUi
}

func (c *ShellCommand) Help() string {
	return strings.TrimSpace(fmt.Sprintf(usage))
}

func (c *ShellCommand) Synopsis() string {
	return synopsys
}

func (c *ShellCommand) Run(args []string) int {
	var username string
	cmdFlags := flag.NewFlagSet("shell", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.StringVarP(&username, "username", "u", "", usernameUsage)

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 2 {
		c.UI.Error(c.Help())
		return 1
	}
	host := args[0]
	containerID := args[1]
	command := getLoginCommand(username)

	// prepare the term
	stdin := c.UI.Reader
	stdout := c.UI.Writer
	pty, ok := stdout.(*os.File)
	if !ok {
		c.UI.Error("nitro shell requires a working terminal")
		return 1
	}

	err = httpshell.Run(host, containerID, command, pty.Fd(), stdin, stdout)
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
