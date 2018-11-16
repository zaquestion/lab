package cmd

import (
	"github.com/spf13/cobra"
)

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: `List and search labels`,
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
		return
	},
}

func init() {
	RootCmd.AddCommand(labelCmd)
}
