package commands

import "github.com/spf13/cobra"

var nitroCmd = &cobra.Command{Use: "nitro"}

func Execute() {
	nitroCmd.AddCommand(loginCmd)
	nitroCmd.Execute()
}
