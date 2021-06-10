package cmd

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mergeImmediate bool

var mrMergeCmd = &cobra.Command{
	Use:   "merge [remote] <id>",
	Short: "Merge an open merge request",
	Long: heredoc.Doc(`
		Merges an open merge request. If the pipeline in the project is
		enabled and is still running for that specific MR, by default,
		this command will sets the merge to only happen when the pipeline
		succeeds.
	`),
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

		opts := gitlab.AcceptMergeRequestOptions{
			MergeWhenPipelineSucceeds: gitlab.Bool(!mergeImmediate),
		}

		err = lab.MRMerge(p.ID, int(id), &opts)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Merge Request !%d merged\n", id)
	},
}

func init() {
	mrMergeCmd.Flags().BoolVarP(&mergeImmediate, "immediate", "i", false, "merge immediately, regardless pipeline results")
	mrCmd.AddCommand(mrMergeCmd)
	carapace.Gen(mrMergeCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
