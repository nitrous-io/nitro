package httpshell

import (
	"fmt"
	"io"
)

// Convenient function to run a remote command inside a container in a specific
// host
func Run(
	host, container, command string,
	fd uintptr,
	stdin io.Reader, stdout io.Writer,
) error {
	s := NewHTTPShell(host, container, command)
	if err := s.RequestShell(); err != nil {
		return fmt.Errorf("RequestShell: %v", err)
	}
	if err := s.Attach(fd, stdin, stdout); err != nil {
		return fmt.Errorf("Shell detached %v", err)
	}
	return nil
}
