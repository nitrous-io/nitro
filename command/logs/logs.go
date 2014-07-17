package logs

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	flag "github.com/ogier/pflag"
)

const (
	followUsage     = "Follow log output"
	timestampsUsage = "Show timestamps"
)

var usage = fmt.Sprintf(`
Usage: nitro logs [HOST] [CONTAINER]

Fetch the logs of the container

Options:

  -f, --follow=""         %s.
  -t, --timestamps=""     %s.
`, followUsage, timestampsUsage)

type Command struct {
	UI *cli.BasicUi
}

func init() {
	command.Register("logs", NewCommand)
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
	var follow bool
	var timestamps bool

	cmdFlags := flag.NewFlagSet("logs", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.BoolVarP(&follow, "follow", "f", false, followUsage)
	cmdFlags.BoolVarP(&timestamps, "timestamps", "t", false, timestampsUsage)

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

	params := url.Values{}
	if follow {
		params.Set("follow", "1")
	}
	if timestamps {
		params.Set("timestamps", "1")
	}

	endpoint := fmt.Sprintf("%s/containers/%s/logs?%s", host, containerID, params.Encode())
	resp, err := http.Get(endpoint)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to fetch container %s logs on host %s, err : %v",
			containerID, host, err))
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.UI.Error(fmt.Sprintf("Failed to fetch container %s logs on host %s, status : %d",
			containerID, host, resp.StatusCode))
		return 1
	}

	if _, err := io.Copy(c.UI.Writer, resp.Body); err != nil {
		c.UI.Error(fmt.Sprintf("Unexpected error when fetching logs: %v", err))
		return 1
	}
	return 0
}

func (c *Command) Help() string {
	return strings.TrimSpace(usage)
}

func (c *Command) Synopsis() string {
	return "Fetch the logs of a container"
}
