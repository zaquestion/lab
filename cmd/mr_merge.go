package cmd

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mergeImmediate bool

var mrMergeCmd = &cobra.Command{
	Use:   "merge [remote] [<MR id or branch>]",
	Short: "Merge an open merge request",
	Long: heredoc.Doc(`
		Merges an open merge request. If the pipeline in the project is
		enabled and is still running for that specific MR, by default,
		this command will sets the merge to only happen when the pipeline
		succeeds.`),
	Example: heredoc.Doc(`
		lab mr merge origin 10
		lab mr merge upstream 11 -i`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsWithGitBranchMR(args)
		if err != nil {
			log.Fatal(err)
		}

		opts := gitlab.AcceptMergeRequestOptions{
			MergeWhenPipelineSucceeds: gitlab.Bool(!mergeImmediate),
		}

		err = lab.MRMerge(rn, int(id), &opts)
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
