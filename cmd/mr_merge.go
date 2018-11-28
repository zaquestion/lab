package cmd

import (
	"log"

	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrMergeCmd = &cobra.Command{
	Use:     "merge [remote] <id>",
	Aliases: []string{"delete"},
	Short:   "Merge an open merge request",
	Long:    `If the pipeline for the mr is still running, lab sets merge on success`,
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

		err = lab.MRMerge(p.ID, int(id))
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	mrMergeCmd.MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	mrMergeCmd.MarkZshCompPositionalArgumentCustom(2, "__lab_completion_merge_request $words[2]")
	mrCmd.AddCommand(mrMergeCmd)
}
