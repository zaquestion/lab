package cmd

import (
	"fmt"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var issueUnsubscribeCmd = &cobra.Command{
	Use:              "unsubscribe [remote] <id>",
	Aliases:          []string{},
	Short:            "Unubscribe from an issue",
	Long:             ``,
	Example:          "lab issue unsubscribe origin 10",
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsRemoteAndID(args)
		if err != nil {
			log.Fatal(err)
		}

		err = lab.IssueUnsubscribe(rn, int(id))
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
