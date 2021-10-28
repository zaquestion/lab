package cmd

import (
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrUnsubscribeCmd = &cobra.Command{
	Use:     "unsubscribe [remote] [<MR id or branch>]",
	Aliases: []string{},
	Short:   "Unubscribe from merge request",
	Example: heredoc.Doc(`
		lab mr unsubscribe 11
		lab mr unsubscribe origin 12`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsWithGitBranchMR(args)
		if err != nil {
			log.Fatal(err)
		}

		err = lab.MRUnsubscribe(rn, int(id))
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
