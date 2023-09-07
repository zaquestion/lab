package cmd

import (
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	"github.com/zaquestion/lab/internal/gitlab"
)

var milestoneBrowseCmd = &cobra.Command{
	Use:     "browse [remote] [<name>]",
	Aliases: []string{"b"},
	Short:   "View milestone in a browser",
	Example: heredoc.Doc(`
		lab milestone browse
		lab milestone browse upstream "my great milestone"`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, name, err := parseArgsRemoteAndProject(args)
		if err != nil {
			log.Fatal(err)
		}

		var milestoneURL string
		if name == "" {
			project, err := gitlab.FindProject(rn)
			if err != nil {
				log.Fatal(err)
			}

			milestoneURL = project.WebURL + "/-/milestones"
		} else {
			name = strings.TrimLeft(name, "%")
			milestone, err := gitlab.MilestoneGet(rn, name)
			if err != nil {
				log.Fatal(err)
			}

			milestoneURL = milestone.WebURL
		}

		err = browse(milestoneURL)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	milestoneCmd.AddCommand(milestoneBrowseCmd)
	carapace.Gen(milestoneCreateCmd).PositionalCompletion(
		action.Remotes(),
		carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			project, _, err := parseArgsRemoteAndProject(c.Args)
			if err != nil {
				return carapace.ActionMessage(err.Error())
			}
			return action.Milestones(project, action.MilestoneOpts{Active: true})
		}),
	)
}
