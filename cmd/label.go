package cmd

import (
	"github.com/spf13/cobra"
)

var labelCmd = &cobra.Command{
	Use:              "label",
	Short:            "List and search labels",
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	RootCmd.AddCommand(labelCmd)
}
