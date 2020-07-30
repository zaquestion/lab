package cmd

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
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
	carapace.Gen(mergeRequestCmd).PositionalCompletion(
		action.Remotes(),
		action.RemoteBranches(0),
	)
}
