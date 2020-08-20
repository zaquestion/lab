package cmd

import (
	"fmt"
	"log"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrCloseCmd = &cobra.Command{
	Use:     "close [remote] <id>",
	Aliases: []string{"delete"},
	Short:   "Close merge request",
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

		err = lab.MRClose(p.ID, int(id))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Merge Request #%d closed\n", id)
	},
}

func init() {
	mrCmd.AddCommand(mrCloseCmd)
	carapace.Gen(mrCloseCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
