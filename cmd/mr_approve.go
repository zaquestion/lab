package cmd

import (
	"fmt"
	"log"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrApproveCmd = &cobra.Command{
	Use:     "approve [remote] <id>",
	Aliases: []string{},
	Short:   "Approve merge request",
	Long:    ``,
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		p, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}

		err = lab.MRApprove(p.ID, int(id))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Merge Request #%d approved\n", id)
	},
}

func init() {
	mrCmd.AddCommand(mrApproveCmd)
	carapace.Gen(mrApproveCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
