package cmd

import (
	"fmt"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrReopenCmd = &cobra.Command{
	Use:              "reopen [remote] [<MR id or branch>]",
	Short:            "Reopen a closed merge request",
	Example:          "lab mr reopen upstream 20",
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsWithGitBranchMR(args)
		if err != nil {
			log.Fatal(err)
		}

		err = lab.MRReopen(rn, int(id))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Merge Request !%d reopened\n", id)
	},
}

func init() {
	mrCmd.AddCommand(mrReopenCmd)
	carapace.Gen(mrReopenCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
