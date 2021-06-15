package cmd

import (
	"fmt"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var issueSubscribeCmd = &cobra.Command{
	Use:              "subscribe [remote] <id>",
	Aliases:          []string{},
	Short:            "Subscribe to an issue",
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsRemoteAndID(args)
		if err != nil {
			log.Fatal(err)
		}

		p, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}

		err = lab.IssueSubscribe(p.ID, int(id))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Subscribed to issue #%d\n", id)
	},
}

func init() {
	issueCmd.AddCommand(issueSubscribeCmd)
	carapace.Gen(issueSubscribeCmd).PositionalCompletion(
		action.Remotes(),
		action.Issues(issueList),
	)
}
