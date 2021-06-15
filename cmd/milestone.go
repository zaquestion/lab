package cmd

import (
	"github.com/spf13/cobra"
)

var milestoneCmd = &cobra.Command{
	Use:              "milestone",
	Short:            "List and search milestones",
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	RootCmd.AddCommand(milestoneCmd)
}
