package cmd

import (
	"fmt"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var issueReopenCmd = &cobra.Command{
	Use:              "reopen [remote] <id>",
	Short:            "Reopen a closed issue",
	Long:             ``,
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsRemoteAndID(args)
		if err != nil {
			log.Fatal(err)
		}

		p, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}

		err = lab.IssueReopen(p.ID, int(id))
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
