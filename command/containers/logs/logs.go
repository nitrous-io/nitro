package logs

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/containers"
	"github.com/nitrous-io/nitro/pkg/nitrousapi"
	"github.com/nitrous-io/nitro/pkg/protoshell"
	"github.com/nitrous-io/nitrous-common/httpclient"
	"github.com/nitrous-io/nitrous-common/wsconn"
	flag "github.com/ogier/pflag"
)

const (
	followUsage     = "Follow log output"
	timestampsUsage = "Show timestamps"
	tailUsage       = "Output specified number of lines at the end of logs: `all` or `<number>`. Default all"
)

var usage = fmt.Sprintf(`
Usage: nitro logs <container slug>

  Fetch the logs of the container with specified slug.

Options:

  -f, --follow=""         %s.
  -t, --timestamps=""     %s.
  -T, --tail=""           %s.
`, followUsage, timestampsUsage, tailUsage)

type Command struct {
	UI     *command.BasicUI
	Client *httpclient.Client
	Server string

	TLSConfig *tls.Config
}

// `logs` command requires user to be logged in first via `nitro login`
func init() {
	command.RegisterAuth("logs", NewCommand)
}

// Create a new command that when run will return the logs of a container
func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI: &command.BasicUI{cli.BasicUi{
			Reader:      os.Stdin,
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		}},
		Server:    command.Env.APIURL,
		Client:    httpclient.New(token),
		TLSConfig: nil,
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

	token, err := containers.GetToken(slug, c.Server, []string{"logs"}, c.Client)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	cr, err := nitrousapi.GetContainer(c.Client, c.Server, slug)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	host := fmt.Sprintf("%s:443", cr.Host.ProxyHostname)
	params := &url.Values{}
	params.Set("follow", strconv.FormatBool(follow))
	params.Set("timestamps", strconv.FormatBool(timestamps))
	params.Set("tail", tail)
	params.Set("token", token)
	params.Set("freighterd_port", strconv.Itoa(int(cr.Host.FreighterdPort)))

	if err := c.logs(host, cr.CID, c.TLSConfig, params, c.UI.Reader, c.UI.Writer); err != nil {
		c.UI.Error(fmt.Sprintf("Exited: %v", err))
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
	endpoint := fmt.Sprintf("%s/api/v0/containers/%s", c.Server, slug)
	resp, err := c.Client.GetAuthed(endpoint, nil)
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

func (c *Command) logs(host, containerID string, tlsConfig *tls.Config, params *url.Values, stdin io.Reader, stdout io.Writer) error {
	wsEndpointURI := fmt.Sprintf("wss://%s/containers/%s/logs?%s", host, containerID, params.Encode())
	dialer := websocket.Dialer{}
	if tlsConfig != nil {
		dialer.TLSClientConfig = tlsConfig
	}
	ws, _, err := dialer.Dial(wsEndpointURI, http.Header{})
	if err != nil {
		return errors.New("Failed to fetch logs from container.")
	}

	conn := wsconn.NewConnection(ws)
	defer conn.Close()

	in := protoshell.NewShellReader(conn, websocket.TextMessage)
	if _, err = io.Copy(stdout, in); err != nil {
		return err
	}
	return nil
}
