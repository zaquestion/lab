package cmd

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrRebaseCmd = &cobra.Command{
	Use:              "rebase [remote] [<MR id or branch>]",
	Short:            "Rebase an open merge request",
	Example:          "lab mr rebase upstream 20",
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsWithGitBranchMR(args)
		if err != nil {
			log.Fatal(err)
		}

		// FIXME use gitlab.RebaseMergeRequestOptions
		err = lab.MRRebase(rn, int(id), nil)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	mrCmd.AddCommand(mrRebaseCmd)
	carapace.Gen(mrRebaseCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
