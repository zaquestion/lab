package cmd

import (
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var milestoneListCmd = &cobra.Command{
	Use:     "list [remote] [search]",
	Aliases: []string{"ls", "search"},
	Short:   "List milestones",
	Example: heredoc.Doc(`
		lab milestone list
		lab milestone list "search term"
		lab milestone list remote "search term"
		lab milestone list upstream -s 'closed'`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, milestoneSearch, err := parseArgsRemoteAndProject(args)
		if err != nil {
			log.Fatal(err)
		}

		milestoneState, _ := cmd.Flags().GetString("state")
		opts := &gitlab.ListMilestonesOptions{
			State: &milestoneState,
		}

		milestoneSearch = strings.ToLower(milestoneSearch)
		if milestoneSearch != "" {
			opts.Search = &milestoneSearch
		}

		milestones, err := lab.MilestoneList(rn, opts)
		if err != nil {
			log.Fatal(err)
		}

		for _, milestone := range milestones {
			description := ""
			if milestone.Description != "" {
				description = " - " + milestone.Description
			}

			fmt.Printf("%s%s\n", milestone.Title, description)
		}
	},
}

func init() {
	milestoneListCmd.Flags().StringP("state", "s", "active", "filter milestones by state (active/closed)")
	milestoneCmd.AddCommand(milestoneListCmd)

	carapace.Gen(milestoneListCmd).FlagCompletion(carapace.ActionMap{
		"state": carapace.ActionValues("active", "closed"),
	})

	carapace.Gen(milestoneListCmd).PositionalCompletion(
		action.Remotes(),
	)
}
