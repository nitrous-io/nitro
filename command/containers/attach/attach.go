package attach

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/pkg/nitrousapi"
	"github.com/nitrous-io/nitrous-common/httpclient"
	flag "github.com/ogier/pflag"
)

var usage = `
Usage: nitro attach [SLUG]

  attach to a running container
`

type Command struct {
	UI     *cli.BasicUi
	Client *httpclient.Client
	Server string
}

func init() {
	command.RegisterAuth("attach", NewCommand)
}

func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI: &cli.BasicUi{
			Reader:      os.Stdin,
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		},
		Server: command.Env.APIURL,
		Client: httpclient.New(token),
	}, nil
}

func (c *Command) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("attach", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		c.UI.Error(c.Help())
		return 1
	}
	slug := args[0]

	cr, err := nitrousapi.GetContainer(c.Client, c.Server, slug)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	host := fmt.Sprintf("%s:%d", cr.Host.IPAddress, cr.Host.FreighterdPort)

	tlsConfig, err := nitrousapi.TLSConfig(cr.Host)
	if err != nil {
		c.UI.Error(fmt.Sprintf("failed to get secure HTTPS config %s", err))
		return 1
	}

	err = c.attach(host, cr.CID, tlsConfig, c.UI.Writer)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Exited: %v", err))
		return 1
	}
	return 0
}

func (c *Command) Help() string {
	return strings.TrimSpace(usage)
}

func (c *Command) Synopsis() string {
	return "Attach to a running container"
}

func (c *Command) attach(host string, containerID string, tlsConfig *tls.Config, stdout io.Writer) error {
	dial, err := tls.Dial("tcp", host, tlsConfig)
	if err != nil {
		return fmt.Errorf("Can't connect to %s", host)
	}

	conn, err := httputil.NewClientConn(dial, nil), nil
	if err != nil {
		return err
	}
	defer conn.Close()

	req, err := http.NewRequest("POST", fmt.Sprintf("/containers/%s/attach", containerID), nil)
	if err != nil {
		return err
	}
	req.Host = host

	resp, _ := conn.Do(req)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to attach to the container. Make sure that it's running.")
	}

	wc, r := conn.Hijack()
	defer wc.Close()

	_, err = io.Copy(stdout, r)
	return err
}
