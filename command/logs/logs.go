package logs

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
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

	cr, err := c.getContainerResp(slug)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	params := url.Values{}
	params.Set("follow", strconv.FormatBool(follow))
	params.Set("timestamps", strconv.FormatBool(timestamps))
	params.Set("tail", tail)

	logsEndpoint := fmt.Sprintf("http://%s:%d/containers/%s/logs?%s",
		cr.Host.IpAddress.String, cr.Host.FreighterdPort.Int64,
		cr.Container.CID, params.Encode())
	resp, err := http.Get(logsEndpoint)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to fetch container %s logs, err : %v", slug, err))
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.UI.Error(fmt.Sprintf("Failed to fetch container %s logs, status : %d", slug, resp.StatusCode))
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
