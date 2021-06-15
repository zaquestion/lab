package cmd

import (
	"github.com/spf13/cobra"
)

var issueCmd = &cobra.Command{
	Use:              "issue",
	Short:            `Describe, list, and create issues`,
	Long:             ``,
	PersistentPreRun: labPersistentPreRun,
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
	issueCmd.Flags().BoolP("list", "l", false, "list issues on a remote")
	issueCmd.Flags().MarkDeprecated("list", "use the \"list\" subcommand instead")
	issueCmd.Flags().BoolP("browse", "b", false, "view issue <id> in a browser")
	issueCmd.Flags().MarkDeprecated("browse", "use the \"browse\" subcommand instead")
	issueCmd.Flags().StringP("close", "d", "", "close issue <id> on remote")
	issueCmd.Flags().MarkDeprecated("close", "use the \"close\" subcommand instead")
	RootCmd.AddCommand(issueCmd)
}
