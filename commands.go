package main

import (
	"os"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"
	"github.com/nitrous-io/nitro/environment"
)

// Commands is the mapping of all the available Nitro commands.
var Commands = map[string]cli.CommandFactory{}

func init() {
	env := environment.BaseEnvironment()
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
