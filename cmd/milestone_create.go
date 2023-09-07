package cmd

import (
	"strconv"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var milestoneCreateCmd = &cobra.Command{
	Use:     "create [remote] <name>",
	Aliases: []string{"add"},
	Short:   "Create a new milestone",
	Example: heredoc.Doc(`
		lab milestone create my-milestone
		lab milestone create upstream 'my title' --description 'Some Description'`),
	PersistentPreRun: labPersistentPreRun,
	Args:             cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rn, title, err := parseArgsRemoteAndProject(args)
		if err != nil {
			log.Fatal(err)
		}

		desc, err := cmd.Flags().GetString("description")
		if err != nil {
			log.Fatal(err)
		}

		opts := gitlab.CreateMilestoneOptions{
			Title:       &title,
			Description: &desc,
		}

		start, err := cmd.Flags().GetString("start")
		if err != nil {
			log.Fatal(err)
		}
		if start != "" {
			start := parseDate(start)
			opts.StartDate = &start
		}

		due, err := cmd.Flags().GetString("due")
		if err != nil {
			log.Fatal(err)
		}
		if due != "" {
			due := parseDate(due)
			opts.DueDate = &due
		}

		err = lab.MilestoneCreate(rn, &opts)

		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	milestoneCreateCmd.Flags().String("description", "", "description of the new milestone")
	milestoneCreateCmd.Flags().String("start", "", "start date for the new milestone (YYYY-MM-DD)")
	milestoneCreateCmd.Flags().String("due", "", "due/end date for the new milestone (YYYY-MM-DD)")

	milestoneCmd.AddCommand(milestoneCreateCmd)
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

func parseDate(date string) gitlab.ISOTime {
	// see also CreatePAT() in internal/gitlab/gitlab.go
	s := strings.Split(date, "-")
	if len(s) != 3 {
		log.Fatal("Incorrect date specified, must be YYYY-MM-DD format")
	}

	year, err := strconv.Atoi(s[0])
	if err != nil {
		log.Fatal("Invalid year specified")
	}
	month, err := strconv.Atoi(s[1])
	if err != nil {
		log.Fatal("Invalid month specified")
	}
	day, err := strconv.Atoi(s[2])
	if err != nil {
		log.Fatal("Invalid day specified")
	}

	loc, _ := time.LoadLocation("UTC")
	return gitlab.ISOTime(
		time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc),
	)
}
