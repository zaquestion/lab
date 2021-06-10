package cmd

import (
	"fmt"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrUnsubscribeCmd = &cobra.Command{
	Use:              "unsubscribe [remote] <id>",
	Aliases:          []string{},
	Short:            "Unubscribe from merge request",
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

		err = lab.MRUnsubscribe(p.ID, int(id))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Unsubscribed from merge request !%d\n", id)
	},
}

func init() {
	mrCmd.AddCommand(mrUnsubscribeCmd)
	carapace.Gen(mrUnsubscribeCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
