package cmd

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var milestoneDeleteCmd = &cobra.Command{
	Use:              "delete [remote] <name>",
	Aliases:          []string{"remove"},
	Short:            "Deletes an existing milestone",
	PersistentPreRun: LabPersistentPreRun,
	Args:             cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rn, name, err := parseArgsRemoteAndProject(args)
		if err != nil {
			log.Fatal(err)
		}

		err = lab.MilestoneDelete(rn, name)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	milestoneCmd.AddCommand(milestoneDeleteCmd)
	carapace.Gen(milestoneDeleteCmd).PositionalCompletion(
		action.Remotes(),
		carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			if project, _, err := parseArgsRemoteAndProject(c.Args); err != nil {
				return carapace.ActionMessage(err.Error())
			} else {
				return action.Milestones(project, action.MilestoneOpts{Active: true})
			}
		}),
	)
}
