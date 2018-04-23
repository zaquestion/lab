package cmd

import (
	"github.com/spf13/cobra"
)

var mergeRequestCmd = &cobra.Command{
	Use:   "merge-request [remote [branch]]",
	Short: mrCreateCmd.Short,
	Long:  mrCreateCmd.Long,
	Args:  mrCreateCmd.Args,
	Run: func(cmd *cobra.Command, args []string) {
		runMRCreate(cmd, args)
	},
}

func init() {
	RootCmd.AddCommand(mergeRequestCmd)
}
