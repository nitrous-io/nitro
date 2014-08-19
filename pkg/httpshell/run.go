package httpshell

import (
	"crypto/tls"
	"fmt"
	"io"
)

// Run executes a remote command inside a container in a specific host
func Run(
	host, container, command string,
	tlsConfig *tls.Config,
	fd uintptr,
	stdin io.Reader, stdout io.Writer,
) error {
	s, err := NewHTTPShell(host, container, command, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to create a http shell")
	}
	if err := s.RequestShell(); err != nil {
		return fmt.Errorf("failed to request the shell %v", err)
	}
	if err := s.Attach(fd, stdin, stdout); err != nil {
		return fmt.Errorf("shell detached %v", err)
	}
	return nil
}
