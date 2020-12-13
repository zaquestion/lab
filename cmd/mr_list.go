package cmd

import (
	"fmt"
	"log"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	"github.com/zaquestion/lab/internal/config"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	mrLabels       []string
	mrState        string
	mrTargetBranch string
	mrNumRet       int
	mrAll          bool
	mrMine         bool
	mrDraft        bool
	mrReady        bool
	assigneeID     *int
	mrAssignee     string
	order          string
	sortedBy       string
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:              "list [remote]",
	Aliases:          []string{"ls"},
	Short:            "List merge requests",
	Long:             ``,
	Args:             cobra.MaximumNArgs(1),
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		mrs, err := mrList(args)
		if err != nil {
			log.Print(err)
			config.UserConfigError()
		}
		for _, mr := range mrs {
			fmt.Printf("#%d %s\n", mr.IID, mr.Title)
		}
	},
}

func mrList(args []string) ([]*gitlab.MergeRequest, error) {
	rn, _, err := parseArgsRemoteAndID(args)
	if err != nil {
		return nil, err
	}

	num := mrNumRet
	if mrAll {
		num = -1
	}

	if mrAssignee != "" {
		_assigneeID, err := lab.UserIDByUserName(mrAssignee)
		if err != nil {
			log.Fatal(err)
		}
		assigneeID = &_assigneeID
	} else if mrMine {
		_assigneeID, err := lab.UserID()
		if err != nil {
			log.Fatal(err)
		}
		assigneeID = &_assigneeID
	}

	orderBy := gitlab.String(order)

	sort := gitlab.String(sortedBy)

	opts := gitlab.ListProjectMergeRequestsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: mrNumRet,
		},
		Labels:       mrLabels,
		State:        &mrState,
		TargetBranch: &mrTargetBranch,
		OrderBy:      orderBy,
		Sort:         sort,
		AssigneeID:   assigneeID,
	}

	if mrDraft && !mrReady {
		opts.WIP = gitlab.String("yes")
	} else if mrReady && !mrDraft {
		opts.WIP = gitlab.String("no")
	}

	return lab.MRList(rn, opts, num)
}

func init() {
	listCmd.Flags().StringSliceVarP(
		&mrLabels, "label", "l", []string{}, "filter merge requests by label")
	listCmd.Flags().StringVarP(
		&mrState, "state", "s", "opened",
		"filter merge requests by state (opened/closed/merged)")
	listCmd.Flags().IntVarP(
		&mrNumRet, "number", "n", 10,
		"number of merge requests to return")
	listCmd.Flags().StringVarP(
		&mrTargetBranch, "target-branch", "t", "",
		"filter merge requests by target branch")
	listCmd.Flags().BoolVarP(&mrAll, "all", "a", false, "list all MRs on the project")
	listCmd.Flags().BoolVarP(&mrMine, "mine", "m", false, "list only MRs assigned to me")
	listCmd.Flags().StringVar(
		&mrAssignee, "assignee", "", "list only MRs assigned to $username")
	listCmd.Flags().StringVar(&order, "order", "updated_at", "display order, updated_at(default) or created_at")
	listCmd.Flags().StringVar(&sortedBy, "sort", "desc", "sort order, desc(default) or asc")
	listCmd.Flags().BoolVarP(&mrDraft, "draft", "", false, "list MRs marked as draft")
	listCmd.Flags().BoolVarP(&mrReady, "ready", "", false, "list MRs not marked as draft")

	mrCmd.AddCommand(listCmd)
	carapace.Gen(listCmd).FlagCompletion(carapace.ActionMap{
		"state": carapace.ActionValues("opened", "closed", "merged"),
	})

	carapace.Gen(listCmd).PositionalCompletion(
		action.Remotes(),
	)
}
