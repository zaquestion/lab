package cmd

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	mrLabels       []string
	mrState        string
	mrTargetBranch string
	mrMilestone    string
	mrNumRet       string
	mrAll          bool
	mrMine         bool
	mrAuthor       string
	mrAuthorID     *int
	mrDraft        bool
	mrReady        bool
	mrConflicts    bool
	mrNoConflicts  bool
	mrExactMatch   bool
	mrAssignee     string
	mrAssigneeID   *int
	mrOrder        string
	mrSortedBy     string
	mrReviewer     string
	mrReviewerID   *int
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list [remote] [search]",
	Aliases: []string{"ls", "search"},
	Short:   "List merge requests",
	Long:    ``,
	Args:    cobra.MaximumNArgs(2),
	Example: `lab mr list
lab mr list "search terms"         # search merge requests for "search terms"
lab mr search "search terms"       # same as above
lab mr list remote "search terms"  # search "remote" for merge requests with "search terms"`,
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		mrs, err := mrList(args)
		if err != nil {
			log.Fatal(err)
		}

		pager := NewPager(cmd.Flags())
		defer pager.Close()

		for _, mr := range mrs {
			fmt.Printf("!%d %s\n", mr.IID, mr.Title)
		}
	},
}

func mrList(args []string) ([]*gitlab.MergeRequest, error) {
	rn, search, err := parseArgsRemoteAndProject(args)
	if err != nil {
		return nil, err
	}

	labels, err := MapLabels(rn, mrLabels)
	if err != nil {
		return nil, err
	}

	num, err := strconv.Atoi(mrNumRet)
	if mrAll || (err != nil) {
		num = -1
	}

	// gitlab lib still doesn't have search by assignee and author username
	// for merge requests, because of that we need to get the ID for both.
	if mrAssignee != "" {
		mrAssigneeID = getUserID(mrAssignee)
		if mrAssigneeID == nil {
			log.Fatal(fmt.Errorf("%s user not found\n", mrAssignee))
		}
	} else if mrMine {
		assigneeID, err := lab.UserID()
		if err != nil {
			log.Fatal(err)
		}
		mrAssigneeID = &assigneeID
	}

	if mrAuthor != "" {
		mrAuthorID = getUserID(mrAuthor)
		if mrAuthorID == nil {
			log.Fatal(fmt.Errorf("%s user not found\n", mrAuthor))
		}
	}

	if mrMilestone != "" {
		milestone, err := lab.MilestoneGet(rn, mrMilestone)
		if err != nil {
			log.Fatal(err)
		}
		mrMilestone = milestone.Title
	}

	if mrReviewer != "" {
		mrReviewerID = getUserID(mrReviewer)
		if mrReviewerID == nil {
			log.Fatal(fmt.Errorf("%s user not found\n", mrReviewer))
		}
	}

	orderBy := gitlab.String(mrOrder)

	sort := gitlab.String(mrSortedBy)

	// if none of the flags are set, return every single MR
	mrCheckConflicts := (mrConflicts || mrNoConflicts)

	opts := gitlab.ListProjectMergeRequestsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: num,
		},
		Labels:                 labels,
		State:                  &mrState,
		TargetBranch:           &mrTargetBranch,
		Milestone:              &mrMilestone,
		OrderBy:                orderBy,
		Sort:                   sort,
		AuthorID:               mrAuthorID,
		AssigneeID:             mrAssigneeID,
		WithMergeStatusRecheck: gitlab.Bool(mrCheckConflicts),
		ReviewerID:             mrReviewerID,
	}

	if mrDraft && !mrReady {
		opts.WIP = gitlab.String("yes")
	} else if mrReady && !mrDraft {
		opts.WIP = gitlab.String("no")
	}

	if mrExactMatch {
		if search == "" {
			return nil, errors.New("Exact match requested, but no search terms provided")
		}
		search = "\"" + search + "\""
	}

	if search != "" {
		opts.Search = &search
	}

	mrs, err := lab.MRList(rn, opts, num)
	if err != nil {
		return mrs, err
	}

	// only return MRs that matches the Conflicts requirement
	if mrCheckConflicts {
		var newMrList []*gitlab.MergeRequest
		for _, mr := range mrs {
			if mr.HasConflicts && mrConflicts {
				newMrList = append(newMrList, mr)
			} else if !mr.HasConflicts && mrNoConflicts {
				newMrList = append(newMrList, mr)
			}
		}
		mrs = newMrList
	}

	return mrs, nil
}

func init() {
	listCmd.Flags().StringSliceVarP(
		&mrLabels, "label", "l", []string{}, "filter merge requests by label")
	listCmd.Flags().StringVarP(
		&mrState, "state", "s", "opened",
		"filter merge requests by state (all/opened/closed/merged)")
	listCmd.Flags().StringVarP(
		&mrNumRet, "number", "n", "10",
		"number of merge requests to return")
	listCmd.Flags().StringVarP(
		&mrTargetBranch, "target-branch", "t", "",
		"filter merge requests by target branch")
	listCmd.Flags().StringVar(
		&mrMilestone, "milestone", "", "list only MRs for the given milestone")
	listCmd.Flags().BoolVarP(&mrAll, "all", "a", false, "list all MRs on the project")
	listCmd.Flags().BoolVarP(&mrMine, "mine", "m", false, "list only MRs assigned to me")
	listCmd.Flags().MarkDeprecated("mine", "use --assignee instead")
	listCmd.Flags().StringVar(&mrAuthor, "author", "", "list only MRs authored by $username")
	listCmd.Flags().StringVar(
		&mrAssignee, "assignee", "", "list only MRs assigned to $username")
	listCmd.Flags().StringVar(&mrOrder, "order", "updated_at", "display order (updated_at/created_at)")
	listCmd.Flags().StringVar(&mrSortedBy, "sort", "desc", "sort order (desc/asc)")
	listCmd.Flags().BoolVarP(&mrDraft, "draft", "", false, "list MRs marked as draft")
	listCmd.Flags().BoolVarP(&mrReady, "ready", "", false, "list MRs not marked as draft")
	listCmd.Flags().SortFlags = false
	listCmd.Flags().BoolVar(&mrNoConflicts, "no-conflicts", false, "list only MRs that can be merged")
	listCmd.Flags().BoolVar(&mrConflicts, "conflicts", false, "list only MRs that cannot be merged")
	listCmd.Flags().BoolVarP(&mrExactMatch, "exact-match", "x", false, "match on the exact (case-insensitive) search terms")
	listCmd.Flags().StringVar(
		&mrReviewer, "reviewer", "", "list only MRs with reviewer set to $username")

	mrCmd.AddCommand(listCmd)
	carapace.Gen(listCmd).FlagCompletion(carapace.ActionMap{
		"label": carapace.ActionMultiParts(",", func(c carapace.Context) carapace.Action {
			if project, _, err := parseArgsRemoteAndProject(c.Args); err != nil {
				return carapace.ActionMessage(err.Error())
			} else {
				return action.Labels(project).Invoke(c).Filter(c.Parts).ToA()
			}
		}),
		"milestone": carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			if project, _, err := parseArgsRemoteAndProject(c.Args); err != nil {
				return carapace.ActionMessage(err.Error())
			} else {
				return action.Milestones(project, action.MilestoneOpts{Active: true})
			}
		}),
		"state": carapace.ActionValues("all", "opened", "closed", "merged"),
	})

	carapace.Gen(listCmd).PositionalCompletion(
		action.Remotes(),
	)
}
