package list

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/tabwriter"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/volumes"
	"github.com/nitrous-io/nitro/consts"
	"github.com/nitrous-io/nitrous-common/httpclient"
)

var usage = command.VolumesListUsage

type Command struct {
	UI     cli.Ui
	Server string
	Client *httpclient.Client
}

type Response struct {
	Volumes []volumes.Volume `json:"volumes"`
}

func init() {
	command.RegisterAuth("volumes", NewCommand)
}

func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI:     command.DefaultUI,
		Server: command.Env.APIURL,
		Client: httpclient.New(token),
	}, nil
}

func (c *Command) Help() string {
	return usage.String()
}

func (c *Command) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("list", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		c.UI.Output(c.Help())
		return 1
	}
	host := args[0]

	c.UI.Output(fmt.Sprintf("Listing volumes on '%s'...", host))
	b, err := c.makeRequest(host)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	err = c.handleResponse(b)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	return 0
}

func (c *Command) Synopsis() string {
	return usage.Synopsis
}

func (c *Command) makeRequest(host string) (io.ReadCloser, error) {
	resp, err := c.Client.GetAuthed(fmt.Sprintf("%s/api/v0/hosts/%s/volumes", c.Server, host), nil)
	if err != nil {
		return nil, err
	}
	err = httpclient.CheckResponse(resp, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (c *Command) handleResponse(body io.ReadCloser) error {
	defer body.Close()
	j := Response{}
	err := json.NewDecoder(body).Decode(&j)
	if err != nil {
		return err
	}
	if len(j.Volumes) == 0 {
		c.UI.Output("No volumes found")
		return nil
	}

	// tabwriter is overkill at present because only one column present, but
	// easy to add in future
	b := &bytes.Buffer{}
	w := &tabwriter.Writer{}
	w.Init(b, consts.MinWidth, consts.TabWidth, consts.Padding, consts.PaddingChar, consts.Flags)
	defer func() {
		w.Flush()
		c.UI.Output(b.String())
	}()

	fmt.Fprintln(w, strings.Join([]string{
		"Slug",
	}, "\t"))
	for _, v := range j.Volumes {
		fmt.Fprintln(w, strings.Join([]string{
			v.Slug,
		}, "\t"))
	}
	return nil
}
