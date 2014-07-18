package command

import (
	"os"

	"github.com/mitchellh/cli"
)

// Commands is the mapping of all the available Nitro commands.
var (
	Commands map[string]cli.CommandFactory
	Env      *Environment
)

const ErrorNotLoggedIn = "Auth token is not provided"

func init() {
	Commands = map[string]cli.CommandFactory{}
	Env = ConfigureEnvironment()
}

var DefaultUI = &BasicUI{cli.BasicUi{
	Reader: os.Stdin,
	Writer: os.Stdout,
}}

func Register(name string, factory cli.CommandFactory) {
	Commands[name] = factory
}
