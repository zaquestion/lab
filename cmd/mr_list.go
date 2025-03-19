package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/pkg/errors"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
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
	mrApprover     string
	mrApproverID   *gitlab.ApproverIDsValue
	mrAssignee     string
	mrAssigneeID   *gitlab.AssigneeIDValue
	mrOrder        string
	mrSortedBy     string
	mrReviewer     string
	mrReviewerID   *gitlab.ReviewerIDValue
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list [remote] [search]",
	Aliases: []string{"ls", "search"},
	Short:   "List merge requests",
	Args:    cobra.MaximumNArgs(2),
	Example: heredoc.Doc(`
		lab mr list
		lab mr list "search terms"
		lab mr list --target-branch main
		lab mr list remote --target-branch main --label my-label
		lab mr list -l bug
		lab mr list -l close'
		lab mr list upstream -n 5
		lab mr list origin -a
		lab mr list --author johndoe
		lab mr list --assignee janedoe
		lab mr list --order created_at
		lab mr list --sort asc
		lab mr list --draft
		lab mr list --ready
		lab mr list --no-conflicts
		lab mr list -x 'test MR'
		lab mr list -r johndoe`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		mrs, err := mrList(args)
		if err != nil {
			log.Fatal(err)
		}

		pager := newPager(cmd.Flags())
		defer pager.Close()

		for _, mr := range mrs {
			fmt.Printf("!%d %s\n", mr.IID, mr.Title)
		}
	},
}

func mrList(args []string) ([]*gitlab.BasicMergeRequest, error) {
	rn, search, err := parseArgsRemoteAndProject(args)
	if err != nil {
		return nil, err
	}

	labels, err := mapLabelsAsLabelOptions(rn, mrLabels)
	if err != nil {
		return nil, err
	}

	num, err := strconv.Atoi(mrNumRet)
	if mrAll || (err != nil) {
		num = -1
	}

	if mrApprover == "any" {
		mrApproverID = gitlab.ApproverIDs(gitlab.UserIDAny)
	} else if mrApprover == "none" {
		mrApproverID = gitlab.ApproverIDs(gitlab.UserIDNone)
	} else if mrApprover != "" {
		approverID := getUserID(mrApprover)
		if approverID == nil {
			log.Fatalf("%s user not found\n", mrApprover)
		}
		mrApproverID = gitlab.ApproverIDs([]int{*approverID})
	}

	// gitlab lib still doesn't have search by assignee and author username
	// for merge requests, because of that we need to get the ID for both.
	if mrAssignee == "any" {
		mrAssigneeID = gitlab.AssigneeID(gitlab.UserIDAny)
	} else if mrAssignee == "none" {
		mrAssigneeID = gitlab.AssigneeID(gitlab.UserIDNone)
	} else if mrAssignee != "" {
		assigneeID := getUserID(mrAssignee)
		if assigneeID == nil {
			log.Fatalf("%s user not found\n", mrAssignee)
		}
		mrAssigneeID = gitlab.AssigneeID(*assigneeID)
	} else if mrMine {
		assigneeID, err := lab.UserID()
		if err != nil {
			log.Fatal(err)
		}
		mrAssigneeID = gitlab.AssigneeID(assigneeID)
	}

	if mrAuthor != "" {
		mrAuthorID = getUserID(mrAuthor)
		if mrAuthorID == nil {
			log.Fatalf("%s user not found\n", mrAuthor)
		}
	}

	if strings.ToLower(mrMilestone) == "any" {
		mrMilestone = "Any"
	} else if strings.ToLower(mrMilestone) == "none" {
		mrMilestone = "None"
	} else if mrMilestone != "" {
		milestone, err := lab.MilestoneGet(rn, mrMilestone)
		if err != nil {
			log.Fatal(err)
		}
		mrMilestone = milestone.Title
	}

	if mrReviewer == "any" {
		mrReviewerID = gitlab.ReviewerID(gitlab.UserIDAny)
	} else if mrReviewer == "none" {
		mrReviewerID = gitlab.ReviewerID(gitlab.UserIDNone)
	} else if mrReviewer != "" {
		reviewerID := getUserID(mrReviewer)
		if reviewerID == nil {
			log.Fatalf("%s user not found\n", mrReviewer)
		}
		mrReviewerID = gitlab.ReviewerID(*reviewerID)
	}

	orderBy := gitlab.String(mrOrder)

	sort := gitlab.String(mrSortedBy)

	// if none of the flags are set, return every single MR
	mrCheckConflicts := (mrConflicts || mrNoConflicts)

	opts := gitlab.ListProjectMergeRequestsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: num,
		},
		Labels:                 &labels,
		State:                  &mrState,
		TargetBranch:           &mrTargetBranch,
		Milestone:              &mrMilestone,
		OrderBy:                orderBy,
		Sort:                   sort,
		AuthorID:               mrAuthorID,
		ApprovedByIDs:          mrApproverID,
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
		var newMrList []*gitlab.BasicMergeRequest
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
		&mrMilestone, "milestone", "", "list only MRs for the given milestone/any/none")
	listCmd.Flags().BoolVarP(&mrAll, "all", "a", false, "list all MRs on the project")
	listCmd.Flags().BoolVarP(&mrMine, "mine", "m", false, "list only MRs assigned to me")
	listCmd.Flags().MarkDeprecated("mine", "use --assignee instead")
	listCmd.Flags().StringVar(&mrAuthor, "author", "", "list only MRs authored by $username")
	listCmd.Flags().StringVar(
		&mrApprover, "approver", "", "list only MRs approved by $username/any/none")
	listCmd.Flags().StringVar(
		&mrAssignee, "assignee", "", "list only MRs assigned to $username/any/none")
	listCmd.Flags().StringVar(&mrOrder, "order", "updated_at", "display order (updated_at/created_at)")
	listCmd.Flags().StringVar(&mrSortedBy, "sort", "desc", "sort order (desc/asc)")
	listCmd.Flags().BoolVarP(&mrDraft, "draft", "", false, "list MRs marked as draft")
	listCmd.Flags().BoolVarP(&mrReady, "ready", "", false, "list MRs not marked as draft")
	listCmd.Flags().SortFlags = false
	listCmd.Flags().BoolVar(&mrNoConflicts, "no-conflicts", false, "list only MRs that can be merged")
	listCmd.Flags().BoolVar(&mrConflicts, "conflicts", false, "list only MRs that cannot be merged")
	listCmd.Flags().BoolVarP(&mrExactMatch, "exact-match", "x", false, "match on the exact (case-insensitive) search terms")
	listCmd.Flags().StringVar(
		&mrReviewer, "reviewer", "", "list only MRs with reviewer set to $username/any/none")

	mrCmd.AddCommand(listCmd)
	carapace.Gen(listCmd).FlagCompletion(carapace.ActionMap{
		"label": carapace.ActionMultiParts(",", func(c carapace.Context) carapace.Action {
			project, _, err := parseArgsRemoteAndProject(c.Args)
			if err != nil {
				return carapace.ActionMessage(err.Error())
			}
			return action.Labels(project).Invoke(c).FilterParts()
		}),
		"milestone": carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			project, _, err := parseArgsRemoteAndProject(c.Args)
			if err != nil {
				return carapace.ActionMessage(err.Error())
			}
			return action.Milestones(project, action.MilestoneOpts{Active: true})
		}),
		"state": carapace.ActionValues("all", "opened", "closed", "merged"),
	})

	carapace.Gen(listCmd).PositionalCompletion(
		action.Remotes(),
	)
}
