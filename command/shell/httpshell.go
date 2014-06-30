package shell

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/nitrous-io/nitrous-common/term"
)

const (
	defaultWidth  = 80
	defaultHeight = 24
)

type shell struct {
	Id string `json:"id"`
}

type HTTPShell struct {
	Id          string `json:"id"`
	host        string
	containerId string
}

func NewHTTPShell(host, containerId string) *HTTPShell {
	return &HTTPShell{host: host, containerId: containerId}
}

// request a new shell id. This will create a remote shell on the specified
// container
func (s *HTTPShell) RequestShell() error {
	endpoint := fmt.Sprintf("/containers/%s/shells", s.containerId)
	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return err
	}
	req.Host = s.host

	c, err := s.getConn()
	if err != nil {
		return err
	}
	defer c.Close()
	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return fmt.Errorf("Failed to request a remote shell %v", resp)
	}

	return json.NewDecoder(resp.Body).Decode(s)
}

// connect stdin and stdout respectively to their remote parts
func (s *HTTPShell) Attach(fd uintptr, stdin io.Reader, stdout io.Writer) error {
	// restore the terminal back to old state when we finish
	var oldState *term.State
	oldState, err := term.SetRawTerminal(fd)
	if err != nil {
		return err
	}
	defer term.RestoreTerminal(fd, oldState)

	endpoint := fmt.Sprintf("/shells/%s/attach", s.Id)
	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return err
	}
	req.Host = s.host

	c, err := s.getConn()
	if err != nil {
		return err
	}
	defer c.Close()
	// we ignore error because the connection will be hijacked
	resp, _ := c.Do(req)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to attach to the remote shell %v", resp)
	}

	// hijack the http . We have the stream writer and reader now
	wc, r := c.Hijack()
	defer wc.Close()

	done := make(chan error, 1)
	go func() {
		if err := s.monitorSize(fd); err != nil {
			done <- err
			return
		}
		_, err := io.Copy(stdout, r)
		done <- err
	}()
	go io.Copy(wc, stdin)

	return <-done
}

func (s *HTTPShell) monitorSize(fd uintptr) error {
	if err := s.resize(fd); err != nil {
		return err
	}
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGWINCH)
	go func() {
		for _ = range sigchan {
			s.resize(fd)
		}
	}()
	return nil
}

// given the current terminal file descriptor, resize the remote terminal to
// match the local terminal size
func (s *HTTPShell) resize(fd uintptr) error {
	w, h := getSize(fd)
	values := make(url.Values)
	values.Set("width", strconv.Itoa(w))
	values.Set("height", strconv.Itoa(h))
	req, err := http.NewRequest("POST",
		fmt.Sprintf("/shells/%s/resize", s.Id),
		bytes.NewBufferString(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Host = s.host

	c, err := s.getConn()
	if err != nil {
		return err
	}
	defer c.Close()
	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("Failed to resize %v %v", s.Id, resp)
	}
	return nil
}

// get a client connection to the remote server. The connection need to
// implements Hijacker
func (s *HTTPShell) getConn() (*httputil.ClientConn, error) {
	dial, err := net.Dial("tcp", s.host)
	if err != nil {
		return nil, fmt.Errorf("Can't connect to %v", s.host)
	}
	return httputil.NewClientConn(dial, nil), nil
}

// get current size of the client terminal
func getSize(fd uintptr) (int, int) {
	ws, err := term.GetWinsize(fd)
	if err != nil {
		return defaultWidth, defaultHeight
	}
	return int(ws.Width), int(ws.Height)
}
