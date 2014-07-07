package run

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/pkg/httpshell"
)

const usage = `
Usage: nitro run [host] [container] [cmd]

  run a command on a remote docker container.
`

func init() {
	command.Register("run", NewCommand)
}

type Command struct {
	UI *cli.BasicUi
}

func NewCommand() (cli.Command, error) {
	return &Command{
		UI: &cli.BasicUi{
			Reader:      os.Stdin,
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		},
	}, nil
}

func (c *Command) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("run", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}
	args = cmdFlags.Args()
	if len(args) < 3 {
		c.UI.Output(c.Help())
		os.Exit(1)
	}
	host := args[0]
	containerID := args[1]
	command := strings.Join(args[2:], " ")
	log.Printf("Running command: %s", command)

	s := httpshell.NewHTTPShell(host, containerID, command)

	if err := s.RequestShell(); err != nil {
		c.UI.Error(fmt.Sprintf("RequestShell: %v", err))
		return 1
	}

	stdin := c.UI.Reader
	stdout := c.UI.Writer
	pty, ok := stdout.(*os.File)
	if !ok {
		c.UI.Error("Requires a working terminal")
		return 1
	}
	if err := s.Attach(pty.Fd(), stdin, stdout); err != nil {
		c.UI.Error(fmt.Sprintf("Detached %v", err))
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
