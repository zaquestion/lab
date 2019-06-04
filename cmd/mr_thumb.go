package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
	zsh "github.com/rsteube/cobra-zsh-gen"
)

var mrThumbCmd = &cobra.Command{
	Use:     "thumb",
	Aliases: []string{},
	Short:   "Thumb operations on merge requests",
	Long:    ``,
}

var mrThumbUpCmd = &cobra.Command{
	Use:     "up [remote] <id>",
	Aliases: []string{},
	Short:   "Thumb up merge request",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		p, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}

		err = lab.MRThumbUp(p.ID, int(id))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Merge Request #%d thumb'd up\n", id)
	},
}

var mrThumbDownCmd = &cobra.Command{
	Use:     "down [remote] <id>",
	Aliases: []string{},
	Short:   "Thumbs down merge request",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		p, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}

		err = lab.MRThumbDown(p.ID, int(id))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Merge Request #%d thumb'd down\n", id)
	},
}

func init() {
	mrCmd.AddCommand(mrThumbCmd)

	zsh.Wrap(mrThumbUpCmd).MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	zsh.Wrap(mrThumbUpCmd).MarkZshCompPositionalArgumentCustom(2, "__lab_completion_merge_request $words[2]")
	mrThumbCmd.AddCommand(mrThumbUpCmd)

	zsh.Wrap(mrThumbDownCmd).MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	zsh.Wrap(mrThumbDownCmd).MarkZshCompPositionalArgumentCustom(2, "__lab_completion_merge_request $words[2]")
	mrThumbCmd.AddCommand(mrThumbDownCmd)
}
