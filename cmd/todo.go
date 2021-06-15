package cmd

import (
	"github.com/spf13/cobra"
)

var todoCmd = &cobra.Command{
	Use:              "todo",
	Short:            "Check out the todo list for MR or issues",
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		if list, _ := cmd.Flags().GetBool("list"); list {
			todoListCmd.Run(cmd, args)
			return
		}
		if done, _ := cmd.Flags().GetBool("done"); done {
			todoDoneCmd.Run(cmd, args)
			return
		}

		if len(args) == 0 || len(args) > 2 {
			cmd.Help()
			return
		}
	},
}

func init() {
	RootCmd.AddCommand(todoCmd)
}
