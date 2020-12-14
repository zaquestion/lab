package cmd

import (
	"fmt"
	"log"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrThumbCmd = &cobra.Command{
	Use:              "thumb",
	Aliases:          []string{},
	Short:            "Thumb operations on merge requests",
	PersistentPreRun: LabPersistentPreRun,
	Long:             ``,
}

var mrThumbUpCmd = &cobra.Command{
	Use:              "up [remote] <id>",
	Aliases:          []string{},
	Short:            "Thumb up merge request",
	Long:             ``,
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

		err = lab.MRThumbUp(p.ID, int(id))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Merge Request !%d thumb'd up\n", id)
	},
}

var mrThumbDownCmd = &cobra.Command{
	Use:     "down [remote] <id>",
	Aliases: []string{},
	Short:   "Thumbs down merge request",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsRemoteAndID(args)
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
		fmt.Printf("Merge Request !%d thumb'd down\n", id)
	},
}

func init() {
	mrCmd.AddCommand(mrThumbCmd)

	mrThumbCmd.AddCommand(mrThumbUpCmd)
	carapace.Gen(mrThumbUpCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)

	mrThumbCmd.AddCommand(mrThumbDownCmd)
	carapace.Gen(mrThumbDownCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
