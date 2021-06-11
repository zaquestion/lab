package cmd

import (
	"fmt"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrSubscribeCmd = &cobra.Command{
	Use:              "subscribe [remote] <id>",
	Aliases:          []string{},
	Short:            "Subscribe to merge request",
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsWithGitBranchMR(args)
		if err != nil {
			log.Fatal(err)
		}

		p, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}

		err = lab.MRSubscribe(p.ID, int(id))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Subscribed to merge request !%d\n", id)
	},
}

func init() {
	mrCmd.AddCommand(mrSubscribeCmd)
	carapace.Gen(mrSubscribeCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
