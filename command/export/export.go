package export

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	flag "github.com/ogier/pflag"
)

const (
	pathUsage = "Path to the exported local archive"
)

var usage = fmt.Sprintf(`
Usage: nitro export [SERVER] [CONTAINER]

  Write the container contents as a tar archive to a local file.

Options:
  
  -p, --path=""		%s.
`, pathUsage)

type Command struct {
	UI cli.Ui
}

func init() {
	command.Register("export", NewCommand)
}

func NewCommand() (cli.Command, error) {
	return &Command{
		UI: command.DefaultUI,
	}, nil
}

// output content of the container's archive into a local file.
func (c *Command) Run(args []string) int {
	var localPath string

	cmdFlags := flag.NewFlagSet("export", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.StringVarP(&localPath, "path", "p", "", pathUsage)

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 2 {
		c.UI.Error("Requires host and container id")
		return 1
	}

	host := args[0]
	containerID := args[1]

	// if we dont specify local path
	if localPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			c.UI.Error("Failed to get current working directory")
			return 1
		}
		localPath = path.Join(cwd, containerID+".tar")
	}

	tf, err := os.OpenFile(localPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to open %s for writing", localPath))
		return 1
	}

	c.UI.Output(fmt.Sprintf("Downloading container %s to %s ...",
		containerID, localPath))
	if err := downloadArchive(c.UI, tf, host, containerID); err != nil {
		c.UI.Error(err.Error())
		return 1
	}
	c.UI.Output(fmt.Sprintf("Downloaded container %s to %s ...",
		containerID, localPath))
	return 0
}

func (c *Command) Help() string {
	return strings.TrimSpace(usage)
}

func (c *Command) Synopsis() string {
	return "Export the container as a tar archive"
}

// Download an exported container archive to localPath. If localPath is empty,
// then we use the current directory to store the archive
func downloadArchive(ui cli.Ui, w io.Writer, host, containerID string) error {
	// get the container archive as a stream
	resp, err := http.Get(fmt.Sprintf("%s/containers/%s/export", host, containerID))
	if err != nil {
		return fmt.Errorf("Failed to export container, err: %v", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("Invalid container: %s", containerID)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to export container, status: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	// save the archive stream into the local archiv
	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("Failed to save container archive, err: %v", err)
	}
	return nil
}

func getLocalArchive(containerID, localPath string) (*os.File, error) {
	// if we dont specify local path
	if localPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, errors.New("Failed to get current working directory")
		}
		localPath = path.Join(cwd, containerID+".tar")
	}

	tf, err := os.Create(localPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to open %s for writing", localPath)
	}
	return tf, nil
}
