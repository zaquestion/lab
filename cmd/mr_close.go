package cmd

import (
	"fmt"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrCloseCmd = &cobra.Command{
	Use:              "close [remote] [<MR id or branch>]",
	Short:            "Close merge request",
	Example:          "lab mr close origin 10",
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsWithGitBranchMR(args)
		if err != nil {
			log.Fatal(err)
		}

		err = lab.MRClose(rn, int(id))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Merge Request !%d closed\n", id)
	},
}

func init() {
	mrCmd.AddCommand(mrCloseCmd)
	carapace.Gen(mrCloseCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
