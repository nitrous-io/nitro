package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/mitchellh/cli"
	nlogger "github.com/nitrous-io/nitrous-common/logger"
)

var version string

var logger nlogger.Logger

func init() {
	logger = nlogger.Logger{}
	l := os.Getenv("NITRO_LOG")
	switch l {
	// TODO support more levels, better yet nitrous-common/logger needs
	// initializer with string arg
	case "DEBUG":
		logger.Level = nlogger.LogLevelDebug
	default:
		logger.Level = nlogger.LogLevelError
	}

	s := os.Getenv("NO_SSL_VERIFY")
	if v, _ := strconv.ParseBool(s); v {
		logger.Warn("SSL cert verification disabled!")
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
}

func main() {
	cli := cli.NewCLI("nitro", version)
	cli.Args = os.Args[1:]
	cli.Commands = Commands
	cli.HelpFunc = helpFunc

	exitCode, err := cli.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "nitro: unexpected error: %v\n", err)
		os.Exit(1)
	}

	os.Exit(exitCode)
}

func helpFunc(commands map[string]cli.CommandFactory) string {
	basicHelp := cli.BasicHelpFunc("nitro")(commands)
	return fmt.Sprintf("nitro is a command-line interface to Nitrous.IO.\n\n%s", basicHelp)
}
