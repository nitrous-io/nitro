package shell

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitrous-common/term"
)

const (
	defaultTerm = "xterm-256color"
)

func init() {
	command.Register("shell", NewShellCommand)
}

func NewShellCommand() (cli.Command, error) {
	return &ShellCommand{
		UI: &cli.BasicUi{
			Reader: os.Stdin,
			Writer: os.Stdout,
		},
	}, nil
}

type ShellCommand struct {
	UI cli.Ui
}

func (c *ShellCommand) Help() string {
	usage := `
Usage: nitro shell <host_name> <container_id>

open a shell to a box on a Nitrous.IO zone
`
	return strings.TrimSpace(fmt.Sprintf(usage))
}

func (c *ShellCommand) Run(args []string) int {
	host := args[0]
	container := args[1]

	log.Println(host, container)

	params := url.Values{}
	params.Set("width", "80")
	params.Set("height", "60")

	// get the shell id
	endpoint := fmt.Sprintf("/containers/%s/shells?%s", container, params.Encode())
	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return 1
	}
	req.Header.Set("Content-Type", "plain/text")
	req.Host = host

	clientconn, err := getConn(host)
	if err != nil {
		return 1
	}
	resp, err := clientconn.Do(req)
	if err != nil {
		return 1
	}

	session := &struct {
		Id string `json:"id"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(session); err != nil {
		return 1
	}

	endpoint = fmt.Sprintf("/shells/%s/attach", session.Id)
	req, err = http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return 1
	}
	req.Host = host

	clientconn, err = getConn(host)
	if err != nil {
		return 1
	}
	defer clientconn.Close()
	resp, _ = clientconn.Do(req)
	if resp.StatusCode != 200 {
		log.Println("invalid response", resp.StatusCode)
		return 1
	}

	// hijack the http connection here. We have the stream now
	// set this stream as the session stdin/stdout/stderr
	rwc, _ := clientconn.Hijack()

	// prepare the term
	w := c.UI.(*cli.BasicUi).Writer
	terminalFd := w.(*os.File).Fd()

	// restore the terminal back to old state when we finish
	var oldState *term.State
	oldState, err = term.SetRawTerminal(terminalFd)
	if err != nil {
		return 1
	}
	defer term.RestoreTerminal(terminalFd, oldState)

	done := make(chan error, 1)
	go func() {
		monitorSize(host, session.Id, terminalFd)
		_, err := io.Copy(c.UI.(*cli.BasicUi).Writer, rwc)
		done <- err
	}()
	go io.Copy(rwc, c.UI.(*cli.BasicUi).Reader)

	<-done
	return 0
}

// get current size of the client terminal
func getSize(fd uintptr) (int, int) {
	ws, err := term.GetWinsize(fd)
	if err != nil {
		// just use 80, 24 as the default
		return 80, 24
	}
	return int(ws.Width), int(ws.Height)
}

func postResize(host, shellId string, fd uintptr) error {
	w, h := getSize(fd)
	values := make(url.Values)
	values.Set("width", strconv.Itoa(w))
	values.Set("height", strconv.Itoa(h))
	req, err := http.NewRequest("POST",
		fmt.Sprintf("/shells/%s/resize", shellId),
		bytes.NewBufferString(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Host = host

	clientconn, err := getConn(host)
	if err != nil {
		return fmt.Errorf("Can't connect to daemon %v", host)
	}
	defer clientconn.Close()
	_, err = clientconn.Do(req)
	return err
}

// monitor size of the current client and post the change to server
func monitorSize(host, shellId string, fd uintptr) error {
	postResize(host, shellId, fd)
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGWINCH)
	go func() {
		for _ = range sigchan {
			postResize(host, shellId, fd)
		}
	}()
	return nil
}

func (c *ShellCommand) Synopsis() string {
	return "Open a shell on Nitrous.IO box"
}

func getConn(host string) (*httputil.ClientConn, error) {
	dial, err := net.Dial("tcp", host)
	if err != nil {
		return nil, fmt.Errorf("Can't connect to containerd daemon %v", host)
	}
	return httputil.NewClientConn(dial, nil), nil
}
