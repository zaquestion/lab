package cmd

import (
	"fmt"
	"log"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var issueUnsubscribeCmd = &cobra.Command{
	Use:              "unsubscribe [remote] <id>",
	Aliases:          []string{},
	Short:            "Unubscribe from issue",
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

		err = lab.IssueUnsubscribe(p.ID, int(id))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Unsubscribed from issue #%d\n", id)
	},
}

func init() {
	issueCmd.AddCommand(issueUnsubscribeCmd)
	carapace.Gen(issueUnsubscribeCmd).PositionalCompletion(
		action.Remotes(),
		action.Issues(issueList),
	)
}
