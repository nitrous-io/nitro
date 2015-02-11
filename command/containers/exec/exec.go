package exec

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/dedicated/nitro/command"
	"github.com/nitrous-io/dedicated/nitro/command/containers"
	"github.com/nitrous-io/dedicated/nitro/pkg/nitrousapi"
	"github.com/nitrous-io/dedicated/nitro/pkg/protoshell"
	"github.com/nitrous-io/nitrous-common/httpclient"
	flag "github.com/ogier/pflag"
)

const (
	nottyUsage = "do not allocate an interactive TTY for the command"
)

var usage = fmt.Sprintf(`
Usage: nitro exec [OPTIONS] [container] [cmd]

  run a command on a container

Options:
  --notty="" %s.
`, nottyUsage)

func init() {
	command.RegisterAuth("exec", NewCommand)
}

type Command struct {
	UI        *cli.BasicUi
	client    *httpclient.Client
	server    string
	tlsConfig *tls.Config
}

func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI: &cli.BasicUi{
			Reader: os.Stdin,
			Writer: os.Stdout,
		},
		server:    command.Env.APIURL,
		client:    httpclient.New(token),
		tlsConfig: nil,
	}, nil
}

func (c *Command) Run(args []string) int {
	var noTTY bool
	cmdFlags := flag.NewFlagSet("exec", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.BoolVar(&noTTY, "notty", false, nottyUsage)

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) < 2 {
		c.UI.Error(c.Help())
		return 1
	}

	slug := args[0]

	cr, err := nitrousapi.GetContainer(c.client, c.server, slug)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}
	if cr.State != "running" {
		c.UI.Error("Requires a running container")
		return 1
	}

	token, err := containers.GetToken(slug, c.server, []string{"shell"}, c.client)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	host := fmt.Sprintf("%s:%d", cr.Host.ProxyHostname, command.FreighterdPort)
	command := strings.Join(args[1:], " ")

	// prepare the term
	stdin, ok := c.UI.Reader.(*os.File)
	if !ok {
		c.UI.Error("Requires a working terminal")
		return 1
	}
	stdout, ok := c.UI.Writer.(*os.File)
	if !ok {
		c.UI.Error("Requires a working terminal")
		return 1
	}

	// construct params header that includes the wanted command
	params := url.Values{}
	params.Set("command", command)
	params.Set("token", token)
	params.Set("binary", "true")
	params.Set("notty", strconv.FormatBool(noTTY))
	endpointURI := fmt.Sprintf("wss://%s/containers/%s/shell?%s", host, cr.CID, params.Encode())

	sh := protoshell.New(endpointURI, c.tlsConfig)
	if err := sh.Run(stdin.Fd(), stdout.Fd(), stdin, stdout); err != nil {
		c.UI.Error(fmt.Sprintf("Exited: %v", err))
		return 1
	}
	return 0
}

func (c *Command) Help() string {
	return strings.TrimSpace(usage)
}

func (c *Command) Synopsis() string {
	return "Run a command on a container"
}
