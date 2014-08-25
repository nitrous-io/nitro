package command

import (
	"bytes"
	"fmt"
	"text/tabwriter"
	"text/template"

	"github.com/nitrous-io/nitro/consts"
)

const UsageTemplate = `
Usage: nitro {{ .Subcommand }}{{ if .Required }} {{ .Required }}{{ end }}{{ if .Options }} [options]{{ end }}

  {{ .Synopsis }}
`

type Option struct {
	Short   string
	Default string
	Usage   string
}

type CommandUsage struct {
	Subcommand   string
	Required     string
	Synopsis     string
	OptionsOrder []string // optional, specify order in which options appear
	Options      map[string]Option
}

func (cu CommandUsage) String() string {
	t := template.Must(template.New("usage").Parse(UsageTemplate))
	b := bytes.NewBufferString("")
	err := t.Execute(b, cu)
	if err != nil {
		panic(err)
	}
	if len(cu.Options) == 0 {
		return b.String()
	}
	b.Write([]byte("\nOptions:\n\n"))
	tw := &tabwriter.Writer{}
	tw.Init(b, consts.MinWidth, consts.TabWidth, consts.Padding, consts.PaddingChar, consts.Flags)

	iter := cu.OptionsOrder
	if len(iter) == 0 {
		for k, _ := range cu.Options {
			iter = append(iter, k)
		}
	}
	for _, optName := range iter {
		opt, ok := cu.Options[optName]
		if !ok {
			panic(fmt.Sprintf("Undefined option %s for %s", optName, cu.Subcommand))
		}
		fmt.Fprintf(tw, "\t-%s, --%s=%s\t%s\n", opt.Short, optName, opt.Default, opt.Usage)
	}
	tw.Flush()
	return b.String()
}

func (cu CommandUsage) OptUsage(name string) string {
	opt, ok := cu.Options[name]
	if !ok {
		panic(fmt.Sprintf("Undefined option %s for %s", name, cu.Subcommand))
	}
	return opt.Usage
}

var ContainerCreateUsage = &CommandUsage{
	Subcommand:   "create",
	Required:     "<host>",
	Synopsis:     "Create a new container",
	OptionsOrder: []string{"slug", "image", "env", "volume", "port-forward"},
	Options: map[string]Option{
		// without fields easier to read, more painful to change... which?
		"slug":         {"s", `""`, "A unique slug for the container"},
		"image":        {Short: "i", Default: `""`, Usage: "An image to use to create the container"},
		"env":          {Short: "e", Default: "[]", Usage: "Set environment variables for the container in the format <key>:<value>. " + MultiargUsage},
		"volume":       {Short: "v", Default: "[]", Usage: "Add data volume so other containers can mount given path. " + MultiargUsage},
		"port-forward": {Short: "p", Default: "[]", Usage: "Forward ports from container to host. Use one of: 'ip:hostPort:containerPort', 'ip::containerPort', or 'hostPort:containerPort' specifications. " + MultiargUsage},
	},
}

var VolumesListUsage = &CommandUsage{
	Subcommand: "volumes",
	Required:   "<host>",
	Synopsis:   "List data volumes on host",
}
