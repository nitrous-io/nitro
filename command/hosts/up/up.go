// TODO: This has been added as a proof of concept for tug, so we can get feedback
// from customers quickly without integrating tug into the nitro binary
package up

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitrous-common/httpclient"
	flag "github.com/ogier/pflag"
)

const verboseUsage = "Verbose Output"
const portUsage = "Forward ports from containerport to localport"
const syncUsage = "Setup two-way sync between localpath and containerpath"

var usage = fmt.Sprintf(`
Usage: nitro up <host>

  Bootstrap a Docker-based project on a host

Options:
  -v, --verbose=false %s
  -p, --port-forward=localport:containerport, %s
  -s, --sync=localpath:containerpath, %s
`, verboseUsage, portUsage, syncUsage)

func init() {
	command.RegisterAuth("up", NewCommand)
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
	var verbose bool
	var portForwards string
	var sync string

	cmdFlags := flag.NewFlagSet("up", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.BoolVarP(&verbose, "verbose", "v", false, verboseUsage)
	cmdFlags.StringVarP(&portForwards, "port-forward", "p", "", portUsage)
	cmdFlags.StringVarP(&sync, "sync", "s", "", syncUsage)

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) < 1 {
		c.UI.Error(c.Help())
		return 1
	}

	host := args[0]
	tugArgs := []string{"start"}
	if verbose {
		tugArgs = append(tugArgs, "-v")
	}

	if portForwards != "" {
		tugArgs = append(tugArgs, "-p", portForwards)
	}

	if sync != "" {
		tugArgs = append(tugArgs, "-s", sync)
	}

	cmd := exec.Command("tug", tugArgs...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("NITRO_HOST=%s", host))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return 1
	}

	return 0
}

func (c *Command) Help() string {
	return strings.TrimSpace(usage)
}

func (c *Command) Synopsis() string {
	return "Bootstrap a Docker-based project on a host"
}
