package cmd

import (
	"github.com/spf13/cobra"
)

// mrCmd represents the mr command
var mrCmd = &cobra.Command{
	Use:   "mr",
	Short: `Describe, list, and create merge requests`,
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if list, _ := cmd.Flags().GetBool("list"); list {
			listCmd.Run(cmd, args)
			return
		}

		if browse, _ := cmd.Flags().GetBool("browse"); browse {
			mrBrowseCmd.Run(cmd, args)
			return
		}

		if id, _ := cmd.Flags().GetString("close"); id != "" {
			mrCloseCmd.Run(cmd, append(args, id))
			return
		}

		if len(args) == 0 || len(args) > 2 {
			cmd.Help()
			return
		}

		mrShowCmd.Run(cmd, args)
	},
}

func init() {
	mrCmd.Flags().BoolP("list", "l", false, "List merge requests on a remote")
	mrCmd.Flags().BoolP("browse", "b", false, "View merge request <id> in a browser")
	mrCmd.Flags().StringP("close", "d", "", "Close merge request <id> on remote")
	RootCmd.AddCommand(mrCmd)
}
