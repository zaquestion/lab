package cmd

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrRebaseCmd = &cobra.Command{
	Use:              "rebase [remote] <id>",
	Short:            "Rebase an open merge request",
	Example:          "lab mr rebase upstream 20",
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsWithGitBranchMR(args)
		if err != nil {
			log.Fatal(err)
		}

		p, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}

		err = lab.MRRebase(p.ID, int(id))
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
