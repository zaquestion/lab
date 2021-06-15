package cmd

import (
	"fmt"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrReopenCmd = &cobra.Command{
	Use:              "reopen [remote] <id>",
	Short:            "Reopen a closed merge request",
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

		err = lab.MRReopen(p.ID, int(id))
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
