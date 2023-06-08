package cmd

import (
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:              "token",
	Short:            `Show, list, create, and revoke personal access tokens`,
	Long:             ``,
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		// Show data about current token
		if show, _ := cmd.Flags().GetBool("show"); show {
			tokenShowCmd.Run(cmd, args)
			return
		}
		// List all tokens
		if list, _ := cmd.Flags().GetBool("list"); list {
			tokenListCmd.Run(cmd, args)
			return
		}
		// Create a token
		if create, _ := cmd.Flags().GetBool("create"); create {
			tokenCreateCmd.Run(cmd, args)
			return
		}
		// Revoke a token
		if revoke, _ := cmd.Flags().GetBool("revoke"); revoke {
			tokenRevokeCmd.Run(cmd, args)
			return
		}

		cmd.Help()
	},
}

func init() {
	tokenCmd.Flags().BoolP("show", "s", false, "show details about current token")
	tokenCmd.Flags().BoolP("list", "l", false, "list all personal access tokens")
	tokenCmd.Flags().BoolP("create", "c", false, "create a personal access tokens")
	tokenCmd.Flags().BoolP("revoke", "r", false, "revoke a personal access tokens")
	RootCmd.AddCommand(tokenCmd)
}
