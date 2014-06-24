package main

import (
	"fmt"
	"os"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/environment"
)

// Commands is the mapping of all the available Nitro commands.
var Commands = map[string]cli.CommandFactory{}

func init() {
	env, err := environment.BaseEnvironmentWithConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error loading config: ", err.Error())
		os.Exit(1)
	}

	defaultUI := &command.BasicUI{cli.BasicUi{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}}

	Commands["login"] = func() (cli.Command, error) {
		return &command.LoginCommand{
			UI:  defaultUI,
			Env: env,
		}, nil
	}
}
