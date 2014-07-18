package list

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/tabwriter"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/login"
	"github.com/nitrous-io/nitro/consts"
	"github.com/nitrous-io/nitrous-common/httpclient"
)

const usage = `
Usage: nitro hosts

  list all hosts
`

type Command struct {
	UI     cli.Ui
	Server string
	Client *httpclient.Client
}

type Host struct {
	Slug  string `json:"slug"`
	State string `json:"state"`
}

func init() {
	command.Register("hosts", NewCommand)
}

func NewCommand() (cli.Command, error) {
	return &Command{
		UI:     command.DefaultUI,
		Server: command.Env.APIURL,
		Client: httpclient.New(command.Env.AuthToken),
	}, nil
}

func (c *Command) Help() string {
	return strings.TrimSpace(usage)
}

func (c *Command) Run(args []string) int {
	if command.Env.AuthToken == "" {
		err := login.PromptLogin()
		if err != nil {
			return 1
		}
		c.Client = httpclient.New(command.Env.AuthToken)
	}

	hosts, err := c.fetchHosts()
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	if len(*hosts) == 0 {
		c.UI.Output("No hosts found")
		return 0
	}

	t := formattedOutput([]string{"Slug", "State"}, hosts)
	c.UI.Output(t)

	return 0
}

func (c *Command) Synopsis() string {
	return "List all hosts"
}

func (c *Command) fetchHosts() (*[]Host, error) {
	resp, err := c.Client.GetAuthed(fmt.Sprintf("%s/api/v0/hosts", c.Server), nil)
	if err != nil {
		return nil, err
	}
	err = httpclient.CheckResponse(resp, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	j := struct {
		Hosts []Host `json:"hosts"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&j)
	if err != nil {
		return nil, err
	}

	return &j.Hosts, nil
}

func formattedOutput(headers []string, hosts *[]Host) string {
	b := &bytes.Buffer{}
	w := &tabwriter.Writer{}

	w.Init(b, consts.MinWidth, consts.TabWidth, consts.Padding, consts.PaddingChar, consts.Flags)
	fmt.Fprintln(w, strings.Join(headers, "\t "))
	for _, r := range *hosts {
		fmt.Fprintln(w, r.Slug, "\t", r.State)
	}
	w.Flush()
	return b.String()
}
