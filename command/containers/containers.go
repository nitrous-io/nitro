package containers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/tabwriter"

	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/consts"
	"github.com/nitrous-io/nitrous-common/httpclient"
)

type Container struct {
	Slug  string `json:"slug"`
	Image string `json:"image"`
	State string `json:"state"`
}

const (
	CmdUsage            = "Command for container to execute"
	EnvUsage            = "Set environment variables for the container in the format <key>=<value>. " + command.MultiargUsage
	VolumesUsage        = "Add data volume so other containers can mount given <path>. " + command.MultiargUsage
	PortForwardsUsage   = "Forward ports from container to host.  Use one of: 'public:hostPort:containerPort', 'private:hostPort:containerPort', 'public::containerPort', or 'private::containerPort' specifications. Publicly forwarded ports will be accessible via the host. " + command.MultiargUsage
	PortForwardAllUsage = "Forward all ports specified by EXPOSE command in Dockerfile"
)

func GetToken(slug, server string, permissions []string, client *httpclient.Client) (string, error) {
	u := fmt.Sprintf("%s/api/v0/containers/%s/token", server, slug)
	p := &url.Values{}
	for _, perm := range permissions {
		p.Add("permission[]", perm)
	}
	resp, err := client.PostFormAuthed(u, p)
	if err != nil {
		return "", fmt.Errorf("Failed to authenticate access to the %q container.", slug)
	}
	if err := httpclient.CheckResponse(resp, http.StatusOK); err != nil {
		return "", fmt.Errorf("You don't have access to the %q container.", slug)
	}
	var j struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&j); err != nil {
		return "", errors.New("An unexpected error has occurred.")
	}
	return j.Token, nil
}

func PrettyPrint(headers []string, containers ...*Container) string {
	b := &bytes.Buffer{}
	w := &tabwriter.Writer{}

	w.Init(b, consts.MinWidth, consts.TabWidth, consts.Padding, consts.PaddingChar, consts.Flags)
	fmt.Fprintln(w, strings.Join(headers, "\t "))
	for _, c := range containers {
		fmt.Fprintln(w,
			c.Slug, "\t",
			c.Image, "\t",
			c.State,
		)
	}
	w.Flush()
	return b.String()
}

func ParseContainerResponse(res *http.Response) (*Container, error) {
	err := httpclient.CheckResponse(res, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	j := struct {
		Container *Container `json:"container"`
	}{}
	err = json.NewDecoder(res.Body).Decode(&j)
	if err != nil {
		return nil, err
	}

	return j.Container, nil
}
