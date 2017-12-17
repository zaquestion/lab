package cmd

import (
	"github.com/spf13/cobra"
)

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: issueShowCmd.Short,
	Long:  issueShowCmd.Long,
	Run: func(cmd *cobra.Command, args []string) {
		if list, _ := cmd.Flags().GetBool("list"); list {
			issueListCmd.Run(cmd, args)
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
	RootCmd.AddCommand(issueCmd)
}
