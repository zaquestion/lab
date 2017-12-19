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

		if len(args) == 0 || len(args) > 2 {
			cmd.Help()
			return
		}

		issueShowCmd.Run(cmd, args)
	},
}

func init() {
	issueCmd.Flags().BoolP("list", "l", false, "list issues")
	issueCmd.Flags().BoolP("browse", "b", false, "browse issues")
	RootCmd.AddCommand(issueCmd)
}
