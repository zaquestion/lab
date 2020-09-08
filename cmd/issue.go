package cmd

import (
	"github.com/spf13/cobra"
)

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: `Describe, list, and create issues`,
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if list, _ := cmd.Flags().GetBool("list"); list {
			issueListCmd.Run(cmd, args)
			return
		}

		if browse, _ := cmd.Flags().GetBool("browse"); browse {
			issueBrowseCmd.Run(cmd, args)
			return
		}

		if id, _ := cmd.Flags().GetString("close"); id != "" {
			issueCloseCmd.Run(cmd, append(args, id))
			return
		}

		if len(args) == 0 || len(args) > 2 {
			cmd.Help()
			return
		}

		issueShowCmd.Run(cmd, args)
	},
}

func init() {
	issueCmd.Flags().BoolP("list", "l", false, "List issues on a remote")
	issueCmd.Flags().BoolP("browse", "b", false, "View issue <id> in a browser")
	issueCmd.Flags().StringP("close", "d", "", "Close issue <id> on remote")
	RootCmd.AddCommand(issueCmd)
}
