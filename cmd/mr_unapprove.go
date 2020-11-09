package cmd

import (
	"fmt"
	"log"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrUnapproveCmd = &cobra.Command{
	Use:              "unapprove [remote] <id>",
	Aliases:          []string{},
	Short:            "Unapprove merge request",
	Long:             ``,
	Args:             cobra.MinimumNArgs(1),
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

		err = lab.MRUnapprove(p.ID, int(id))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Merge Request #%d Unapproved\n", id)
	},
}

func init() {
	mrCmd.AddCommand(mrUnapproveCmd)
	carapace.Gen(mrUnapproveCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
