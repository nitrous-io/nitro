package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/command"

	// import all individual commands here so they register themselves in init()
	_ "github.com/nitrous-io/nitro/command/containers/attach"
	_ "github.com/nitrous-io/nitro/command/containers/commit"
	_ "github.com/nitrous-io/nitro/command/containers/create"
	_ "github.com/nitrous-io/nitro/command/containers/destroy"
	_ "github.com/nitrous-io/nitro/command/containers/inspect"
	_ "github.com/nitrous-io/nitro/command/containers/list"
	_ "github.com/nitrous-io/nitro/command/containers/logs"
	_ "github.com/nitrous-io/nitro/command/containers/reconfig"
	_ "github.com/nitrous-io/nitro/command/containers/restart"
	_ "github.com/nitrous-io/nitro/command/containers/run"
	_ "github.com/nitrous-io/nitro/command/containers/shell"
	_ "github.com/nitrous-io/nitro/command/containers/start"
	_ "github.com/nitrous-io/nitro/command/containers/stop"
	_ "github.com/nitrous-io/nitro/command/hosts/create"
	_ "github.com/nitrous-io/nitro/command/hosts/destroy"
	_ "github.com/nitrous-io/nitro/command/hosts/inspect"
	_ "github.com/nitrous-io/nitro/command/hosts/list"
	_ "github.com/nitrous-io/nitro/command/hosts/restart"
	_ "github.com/nitrous-io/nitro/command/hosts/start"
	_ "github.com/nitrous-io/nitro/command/hosts/stop"
	_ "github.com/nitrous-io/nitro/command/hosts/up"
	_ "github.com/nitrous-io/nitro/command/images/ibuild"
	_ "github.com/nitrous-io/nitro/command/images/list"
	_ "github.com/nitrous-io/nitro/command/images/pull"
	_ "github.com/nitrous-io/nitro/command/images/push"
	_ "github.com/nitrous-io/nitro/command/images/remove"
	_ "github.com/nitrous-io/nitro/command/images/tag"
	_ "github.com/nitrous-io/nitro/command/keys/add"
	_ "github.com/nitrous-io/nitro/command/keys/clear"
	_ "github.com/nitrous-io/nitro/command/keys/list"
	_ "github.com/nitrous-io/nitro/command/keys/remove"
	_ "github.com/nitrous-io/nitro/command/login"
	_ "github.com/nitrous-io/nitro/command/logout"
	_ "github.com/nitrous-io/nitro/command/plans"
	_ "github.com/nitrous-io/nitro/command/regions"
	_ "github.com/nitrous-io/nitro/command/registry/list"
	_ "github.com/nitrous-io/nitro/command/registry/login"
	_ "github.com/nitrous-io/nitro/command/registry/logout"
	_ "github.com/nitrous-io/nitro/command/volumes/create"
	_ "github.com/nitrous-io/nitro/command/volumes/inspect"
	_ "github.com/nitrous-io/nitro/command/volumes/list"
	_ "github.com/nitrous-io/nitro/command/volumes/remove"
)

var (
	version          string // current nitro version
	freighterVersion string // current freighter version that works with current nitro
	nitrousVersion   string // current nitrous version that works with current nitro
)

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
			fmt.Fprintf(os.Stderr, "nitro: requires a logged in user. Run `nitro login` first\n")
		default:
			fmt.Fprintf(os.Stderr, "nitro: unexpected error: %v\n", err)
		}
		os.Exit(1)
	}
	os.Exit(exitCode)
}

// helpFunc iterates through the list of commands and print their synopsis
// It ignores the auth error for commands that require authentication token
func helpFunc(cmds map[string]cli.CommandFactory) string {
	var buf bytes.Buffer
	buf.WriteString("nitro is a command-line interface to Nitrous.IO\n\n")
	buf.WriteString("usage: nitro [--version] [--help] <command> [<args>]\n\n")
	if command.NotLoggedIn() {
		buf.WriteString("Available commands (login to access more commands):\n")
	} else {
		buf.WriteString("Available commands:\n")
	}

	// Get the list of keys so we can sort them, and also get the maximum
	// key length so they can be aligned properly.
	keys := make([]string, 0, len(cmds))
	maxKeyLen := 0
	for key, _ := range cmds {
		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}

		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		cmdFn, ok := cmds[key]
		if !ok {
			// This should never happen since we JUST built the list of
			// keys.
			panic("command not found: " + key)
		}

		cmd, err := cmdFn()
		// we ignore missing auth token error for commands that requires auth
		if err != nil {
			if err != command.ErrMissingAuthToken {
				log.Printf("[ERR] cli: Command '%s' failed to load: %s", key, err)
			}
			continue
		}

		key = fmt.Sprintf("%s%s", key, strings.Repeat(" ", maxKeyLen-len(key)))
		buf.WriteString(fmt.Sprintf("    %s    %s\n", key, cmd.Synopsis()))
	}

	return buf.String()
}
