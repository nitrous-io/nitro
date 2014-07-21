package logs

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/freighter/freighterclient"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/pkg/nitrousapi"
	"github.com/nitrous-io/nitrous-common/httpclient"
	flag "github.com/ogier/pflag"
)

const (
	followUsage     = "Follow log output"
	timestampsUsage = "Show timestamps"
	tailUsage       = "Output specified number of lines at the end of logs: `all` or `<number>`. Default all"
)

var usage = fmt.Sprintf(`
Usage: nitro logs [SLUG]

Fetch the logs of the container with specified slug.

Options:

  -f, --follow=""         %s.
  -t, --timestamps=""     %s.
  -T, --tail=""           %s.
`, followUsage, timestampsUsage, tailUsage)

type Command struct {
	UI     *cli.BasicUi
	client *httpclient.Client
	server string
}

// `logs` command requires user to be logged in first via `nitro login`
func init() {
	command.RegisterAuth("logs", NewCommand)
}

// Create a new command that when run will return the logs of a container
func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI: &cli.BasicUi{
			Reader:      os.Stdin,
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		},
		server: command.Env.APIURL,
		client: httpclient.New(token),
	}, nil
}

func (c *Command) Run(args []string) int {
	var follow bool
	var timestamps bool
	var tail string

	cmdFlags := flag.NewFlagSet("logs", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.BoolVarP(&follow, "follow", "f", false, followUsage)
	cmdFlags.BoolVarP(&timestamps, "timestamps", "t", false, timestampsUsage)
	cmdFlags.StringVarP(&tail, "tail", "T", "all", tailUsage)

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

	host := &freighterclient.Host{
		Host:   cr.Host.IPAddress,
		Port:   int(cr.Host.FreighterdPort),
		Secure: false,
	}
	client := freighterclient.NewClient(nil)

	err = client.Logs(host, cr.CID, strconv.FormatBool(follow),
		strconv.FormatBool(timestamps), tail, c.UI.Writer)
	if err != nil {
		c.UI.Error(err.Error())
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
