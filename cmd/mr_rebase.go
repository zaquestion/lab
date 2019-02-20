package cmd

import (
	"log"

	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrRebaseCmd = &cobra.Command{
	Use:     "rebase [remote] <id>",
	Aliases: []string{"delete"},
	Short:   "Rebase an open merge request",
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

		err = lab.MRRebase(p.ID, int(id))
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	mrRebaseCmd.MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	mrRebaseCmd.MarkZshCompPositionalArgumentCustom(2, "__lab_completion_merge_request $words[2]")
	mrCmd.AddCommand(mrRebaseCmd)
}
