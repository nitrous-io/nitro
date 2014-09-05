package ibuild

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/pkg/randomnamegenerator"
	"github.com/nitrous-io/nitrous-common/httpclient"
	flag "github.com/ogier/pflag"
)

const slugUsage = "A slug for the image (automatically generated if omitted)"

var usage = fmt.Sprintf(`
Usage: nitro images:build <host> <path to Dockerfile or tarball with Dockerfile>

  Build image on host. If a tarball is provided, it must have a Dockerfile in the archive root.

Options:

  -s, --slug=""   %s
`, slugUsage)

var (
	errInvalidTar = fmt.Errorf("Invalid tar file: must contain a file named 'Dockerfile' in archive root")
)

type Command struct {
	UI     command.UI
	Server string
	Client *httpclient.Client
}

func init() {
	command.RegisterAuth("images:build", NewCommand)
}

func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI:     command.DefaultUI,
		Server: command.Env.APIURL,
		Client: httpclient.New(token),
	}, nil
}

func (c *Command) Help() string {
	return strings.TrimSpace(usage)
}

func (c *Command) Run(args []string) int {
	var slug string
	cmdFlags := flag.NewFlagSet("images:build", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.UI.Output(c.Help()) }
	cmdFlags.StringVarP(&slug, "slug", "s", "", slugUsage)

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 2 {
		c.UI.Output(c.Help())
		return 1
	}

	host, fpath := args[0], args[1]
	if slug == "" {
		slug = randomnamegenerator.Generate("image")
	}

	c.UI.Output(fmt.Sprintf("Building image on '%s' from %s ...", host, fpath))

	if err := c.makeRequest(host, fpath, slug); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.UI.Output(fmt.Sprintf("Built image '%s' on '%s'", slug, host))
	return 0
}

func (c *Command) Synopsis() string {
	return "Build image on a host"
}

func (c *Command) makeRequest(host, fpath, slug string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	f, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	s, err := f.Stat()
	if err != nil {
		return err
	}

	part, err := writer.CreateFormFile("file", s.Name())
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, bytes.NewBuffer(b)); err != nil {
		return err
	}
	if err := writer.WriteField("slug", slug); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	path := fmt.Sprintf("%s/api/v0/hosts/%s/images", c.Server, host)
	req, err := http.NewRequest("POST", path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := c.Client.DoAuthed(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return httpclient.CheckResponse(resp, http.StatusOK)
}
