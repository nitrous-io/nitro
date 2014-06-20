package main

import (
	"fmt"
	"os"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"

	// import all individual commands here so they register themselves in init()
	_ "github.com/nitrous-io/nitro/command/login"
	_ "github.com/nitrous-io/nitro/command/logout"
)

var version string

func main() {
	cli := cli.NewCLI("nitro", version)
	cli.Args = os.Args[1:]
	cli.Commands = command.Commands
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
