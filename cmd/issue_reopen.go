package cmd

import (
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var issueReopenCmd = &cobra.Command{
	Use:   "reopen [remote] <id>",
	Short: "Reopen a closed issue",
	Example: heredoc.Doc(`
		lab issue reopen 1
		lab issue reopen upstream 2`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsRemoteAndID(args)
		if err != nil {
			log.Fatal(err)
		}

		err = lab.IssueReopen(rn, int(id))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Issue #%d reopened\n", id)
	},
}

func init() {
	issueCmd.AddCommand(issueReopenCmd)
	carapace.Gen(mrReopenCmd).PositionalCompletion(
		action.Remotes(),
		action.Issues(issueList),
	)
}
