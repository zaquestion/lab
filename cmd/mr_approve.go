package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
	zsh "github.com/rsteube/cobra-zsh-gen"
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
	zsh.Wrap(mrApproveCmd).MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	zsh.Wrap(mrApproveCmd).MarkZshCompPositionalArgumentCustom(2, "__lab_completion_merge_request $words[2]")
	mrCmd.AddCommand(mrApproveCmd)
}
