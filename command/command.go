package command

import (
	"errors"
	"os"

	"github.com/mitchellh/cli"
)

// Commands is the mapping of all the available Nitro commands.
var (
	Commands map[string]cli.CommandFactory
	Env      *Environment

	ErrMissingAuthToken = errors.New("Missing Auth Token")
)

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

// register a command that requires user to be authenticated
type authCommandFactory func(token string) (cli.Command, error)

func RegisterAuth(name string, factory authCommandFactory) {
	Commands[name] = func() (cli.Command, error) {
		if Env.AuthToken == "" {
			return nil, ErrMissingAuthToken
		}
		return factory(Env.AuthToken)
	}
}
