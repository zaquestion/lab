package cmd

import (
	"log"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrMergeCmd = &cobra.Command{
	Use:              "merge [remote] <id>",
	Short:            "Merge an open merge request",
	Long:             `If the pipeline for the mr is still running, lab sets merge on success`,
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsWithGitBranchMR(args)
		if err != nil {
			log.Fatal(err)
		}

		p, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}

		err = lab.MRMerge(p.ID, int(id))
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	mrCmd.AddCommand(mrMergeCmd)
	carapace.Gen(mrMergeCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
