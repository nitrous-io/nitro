package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	username string
	password string

	loginCmd = &cobra.Command{
		Use:   "login",
		Short: "Login to Nitrous.IO",
		Long:  "login authenticates with your Nitrous.IO account",
	}
)

func init() {
	loginCmd.Flags().StringVarP(&username, "user", "u", "", "Nitrous.IO username")
	loginCmd.Flags().StringVarP(&password, "pass", "p", "", "Nitrous.IO password")
	loginCmd.Run = login
}

func login(cmd *cobra.Command, args []string) {
	if !cmd.Flags().Lookup("user").Changed {
		fmt.Printf("Username: ")
		reader := bufio.NewReader(os.Stdin)
		username, _ = reader.ReadString('\n')
	}
	if !cmd.Flags().Lookup("pass").Changed {
		// TODO Don't echo the password.
		fmt.Printf("Password: ")
		reader := bufio.NewReader(os.Stdin)
		password, _ = reader.ReadString('\n')
	}
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	fmt.Printf("Logging in with '%v', '%v'\n", username, password)
}
