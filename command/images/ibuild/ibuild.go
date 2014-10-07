package ibuild

import (
	"archive/tar"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/pkg/archive"
	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/command/images"
	"github.com/nitrous-io/nitro/pkg/buildcontext"
	"github.com/nitrous-io/nitro/pkg/randomnamegenerator"
	"github.com/nitrous-io/nitrous-common/httpclient"
	"github.com/nitrous-io/nitrous-common/isarchive"
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
	UI     *cli.BasicUi
	Server string
	Client *httpclient.Client
}

func init() {
	command.RegisterAuth("images:build", NewCommand)
}

func NewCommand(token string) (cli.Command, error) {
	return &Command{
		UI:     &cli.BasicUi{Reader: os.Stdin, Writer: os.Stdout},
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

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 2 {
		c.UI.Output(c.Help())
		return 1
	}

	stdout, ok := c.UI.Writer.(*os.File)
	if !ok {
		c.UI.Error("Requires a working terminal")
		return 1
	}

	host, fpath := args[0], args[1]
	if slug == "" {
		slug = randomnamegenerator.Generate("image")
	}
	c.UI.Output(fmt.Sprintf("Building image on '%s' from %s ...", host, fpath))

	if err := c.buildImage(host, fpath, slug, stdout, stdout.Fd()); err != nil {
		c.UI.Error(fmt.Sprintf("images:build %s %s failed. Error: %s", host, fpath, err))
		return 1
	}

	c.UI.Output(fmt.Sprintf("Built image '%s' on '%s'", slug, host))
	return 0
}

func (c *Command) Synopsis() string {
	return "Build image on a host"
}

func (c *Command) buildImage(host, fpath, slug string, out io.Writer, termFd uintptr) error {
	var err error
	var body io.Reader

	contentType := "application/x-www-form-urlencoded"
	params := url.Values{
		"slug": []string{slug},
	}

	// check if the given path is available locally
	st, err := os.Stat(fpath)
	if err == nil {
		var r io.Reader
		if st.IsDir() {
			if r, err = wrapBuildContext(fpath); err != nil {
				return err
			}
		} else {
			f, err := os.Open(fpath)
			if err != nil {
				return err
			}
			defer f.Close()
			if r, err = ensureArchive(f); err != nil {
				return err
			}
		}

		var pw *io.PipeWriter
		body, pw = io.Pipe()
		writer := multipart.NewWriter(pw)
		// override the default content-type
		contentType = writer.FormDataContentType()

		go func() {
			defer pw.Close()
			errCh := uploadBuildContext(writer, params, r)
			if err := <-errCh; err != nil {
				log.Printf(err.Error())
			}
			return
		}()
	} else if isRemoteURL(fpath) {
		params.Set("remote", fpath)
		body = bytes.NewBufferString(params.Encode())
	} else {
		return errors.New("path is invalid - requires a path to a local file/directory or a remote URL")
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v0/hosts/%s/images", c.Server, host), body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := c.Client.DoAuthed(req)
	if err != nil {
		return err
	}
	if err = httpclient.CheckResponse(resp, http.StatusOK); err != nil {
		return err
	}
	defer resp.Body.Close()

	if err = images.DisplayJSONMessagesStream(resp.Body, out, termFd, true); err != nil {
		return fmt.Errorf("failed to display build progress: %s", err)
	}
	return nil
}

func uploadBuildContext(writer *multipart.Writer, params url.Values, in io.Reader) chan error {
	errChan := make(chan error, 1)

	for k, v := range params {
		if err := writer.WriteField(k, v[0]); err != nil {
			errChan <- err
			return errChan
		}
	}

	part, err := writer.CreateFormFile("file", "buildcontext.tar")
	if err != nil {
		errChan <- err
		return errChan
	}
	if _, err := io.Copy(part, in); err != nil {
		errChan <- err
		return errChan
	}

	errChan <- writer.Close()
	return errChan
}

func isRemoteURL(s string) bool {
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "git://") ||
		strings.HasPrefix(s, "git@") ||
		strings.HasPrefix(s, "github.com/") ||
		strings.HasSuffix(s, ".git")
}

// wrapBuildContext will traverse the directory and create a tarball containing its files
func wrapBuildContext(src string) (io.Reader, error) {
	filename := path.Join(src, "Dockerfile")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("wrap build context: no Dockerfile found in %s", src)
	}

	var excludes []string
	ignore, err := ioutil.ReadFile(path.Join(src, ".dockerignore"))
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("wrap build context: error reading .dockerignore: '%s'", err)
	}
	for _, pattern := range strings.Split(string(ignore), "\n") {
		ok, err := filepath.Match(pattern, "Dockerfile")
		if err != nil {
			return nil, fmt.Errorf("wrap build context: bad .dockerignore pattern: '%s', error: %s", pattern, err)
		}
		if ok {
			return nil, fmt.Errorf("wrap build context: dockerfile was excluded by .dockerignore pattern '%s'", pattern)
		}
		excludes = append(excludes, pattern)
	}

	// check if a given context directory is valid
	// ie. has permission to access all children
	if err := buildcontext.ValidateDirectory(src, excludes); err != nil {
		return nil, err
	}

	// create tar with options
	return archive.TarWithOptions(src, &archive.TarOptions{
		Compression: archive.Uncompressed,
		Excludes:    excludes,
	})
}

// ensureArchive returns a reader to path if path is a tarball,
// or wraps path in a tarball if path is to a Dockerfile
func ensureArchive(f *os.File) (io.Reader, error) {
	bufRdr := bufio.NewReader(f)
	header, err := bufRdr.Peek(isarchive.MaxHeaderLength)
	// Peek can return err even if header is read and read bytes is shorter than requested
	if err != nil && header == nil {
		return nil, err
	}
	if isarchive.IsArchive(header) {
		return bufRdr, nil
	} else if filepath.Base(f.Name()) == "Dockerfile" {
		return makeArchiveFromDockerfile(f)
	}

	return nil, fmt.Errorf("provided file should be a Dockerfile or a valid archive")
}

// takes a buffered Dockerfile and returns a tar Reader
func makeArchiveFromDockerfile(dockerfile *os.File) (io.Reader, error) {
	stat, err := dockerfile.Stat()
	if err != nil {
		return nil, err
	}

	// reset the file to read from beginning
	_, err = dockerfile.Seek(0, os.SEEK_SET)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)
	tw := tar.NewWriter(buf)

	now := time.Now()
	hdr := &tar.Header{
		Name:       "Dockerfile",
		Size:       stat.Size(),
		ModTime:    now,
		AccessTime: now,
		ChangeTime: now,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := io.Copy(tw, dockerfile); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf, nil
}
