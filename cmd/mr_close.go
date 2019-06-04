package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
	zsh "github.com/rsteube/cobra-zsh-gen"
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
	zsh.Wrap(mrCloseCmd).MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	zsh.Wrap(mrCloseCmd).MarkZshCompPositionalArgumentCustom(2, "__lab_completion_merge_request $words[2]")
	mrCmd.AddCommand(mrCloseCmd)
}
