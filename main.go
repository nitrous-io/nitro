package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"

	// import all individual commands here so they register themselves in init()
	_ "github.com/nitrous-io/nitro/command/containers/create"
	_ "github.com/nitrous-io/nitro/command/containers/destroy"
	_ "github.com/nitrous-io/nitro/command/containers/inspect"
	_ "github.com/nitrous-io/nitro/command/containers/list"
	_ "github.com/nitrous-io/nitro/command/containers/restart"
	_ "github.com/nitrous-io/nitro/command/containers/start"
	_ "github.com/nitrous-io/nitro/command/containers/stop"
	_ "github.com/nitrous-io/nitro/command/hosts/create"
	_ "github.com/nitrous-io/nitro/command/hosts/destroy"
	_ "github.com/nitrous-io/nitro/command/hosts/list"
	_ "github.com/nitrous-io/nitro/command/login"
	_ "github.com/nitrous-io/nitro/command/logout"
	_ "github.com/nitrous-io/nitro/command/logs"
	_ "github.com/nitrous-io/nitro/command/shell"
)

var version string

func main() {
	s := os.Getenv("NO_SSL_VERIFY")
	if v, _ := strconv.ParseBool(s); v {
		fmt.Println("SSL cert verification disabled!")
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	cli := cli.NewCLI("nitro", version)
	cli.Args = os.Args[1:]
	cli.Commands = command.Commands
	cli.HelpFunc = helpFunc

	exitCode, err := cli.Run()
	if err != nil {
		switch err {
		case command.ErrMissingAuthToken:
			fmt.Fprintf(os.Stderr, "nitro: requires a logged in user. Maybe you want nitro login first\n")
		default:
			fmt.Fprintf(os.Stderr, "nitro: unexpected error: %v\n", err)
		}
		os.Exit(1)
	}
	os.Exit(exitCode)
}

func helpFunc(commands map[string]cli.CommandFactory) string {
	basicHelp := cli.BasicHelpFunc("nitro")(commands)
	return fmt.Sprintf("nitro is a command-line interface to Nitrous.IO.\n\n%s", basicHelp)
}
