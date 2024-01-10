package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/pkg/errors"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	issueLabels     []string
	issueMilestone  string
	issueState      string
	issueSearch     string
	issueNumRet     string
	issueAll        bool
	issueExactMatch bool
	issueAssignee   string
	issueAssigneeID *gitlab.AssigneeIDValue
	issueAuthor     string
	issueAuthorID   *int
	issueOrder      string
	issueSortedBy   string
)

var issueListCmd = &cobra.Command{
	Use:     "list [remote] [search]",
	Aliases: []string{"ls", "search"},
	Short:   "List issues",
	Example: heredoc.Doc(`
		lab issue list
		lab issue list "search terms"
		lab issue list origin "search terms"
		lab issue list origin --all
		lab issue list origin --assignee johndoe
		lab issue list upstream --author janedoe
		lab issue list upstream -x "An Issue with Abc"
		lab issue list upstream -l "new_bug"
		lab issue list upstream --milestone "week 22"
		lab issue list remote -n "10"
		lab issue list remote --order "created_at"
		lab issue list remote --sort "asc"
		lab issue list remote --state "closed"`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		issues, err := issueList(args)
		if err != nil {
			log.Fatal(err)
		}

		pager := newPager(cmd.Flags())
		defer pager.Close()

		for _, issue := range issues {
			fmt.Printf("#%d %s\n", issue.IID, issue.Title)
		}
	},
}

func issueList(args []string) ([]*gitlab.Issue, error) {
	rn, search, err := parseArgsRemoteAndProject(args)
	if err != nil {
		return nil, err
	}
	issueSearch = search

	labels, err := mapLabelsAsLabelOptions(rn, issueLabels)
	if err != nil {
		return nil, err
	}

	if strings.ToLower(issueMilestone) == "any" {
		issueMilestone = "Any"
	} else if strings.ToLower(issueMilestone) == "none" {
		issueMilestone = "None"
	} else if issueMilestone != "" {
		milestone, err := lab.MilestoneGet(rn, issueMilestone)
		if err != nil {
			return nil, err
		}
		issueMilestone = milestone.Title
	}

	num, err := strconv.Atoi(issueNumRet)
	if issueAll || (err != nil) {
		num = -1
	}

	// gitlab lib still doesn't have search by author username for issues,
	// because of that we need to get user's ID for both assignee and
	// author.
	if issueAuthor != "" {
		issueAuthorID = getUserID(issueAuthor)
		if issueAuthorID == nil {
			log.Fatalf("%s user not found\n", issueAuthor)
		}
	}

	if issueAssignee == "any" {
		issueAssigneeID = gitlab.AssigneeID(gitlab.UserIDAny)
	} else if issueAssignee == "none" {
		issueAssigneeID = gitlab.AssigneeID(gitlab.UserIDNone)
	} else if issueAssignee != "" {
		assigneeID := getUserID(issueAssignee)
		if assigneeID == nil {
			log.Fatalf("%s user not found\n", issueAssignee)
		}
		issueAssigneeID = gitlab.AssigneeID(*assigneeID)
	}

	orderBy := gitlab.String(issueOrder)

	sort := gitlab.String(issueSortedBy)

	opts := gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: num,
		},
		Labels:     &labels,
		Milestone:  &issueMilestone,
		State:      &issueState,
		OrderBy:    orderBy,
		Sort:       sort,
		AuthorID:   issueAuthorID,
		AssigneeID: issueAssigneeID,
	}

	if issueExactMatch {
		if issueSearch == "" {
			return nil, errors.New("Exact match requested, but no search terms provided")
		}
		issueSearch = "\"" + issueSearch + "\""
	}

	if issueSearch != "" {
		opts.Search = &issueSearch
	}

	return lab.IssueList(rn, opts, num)
}

func init() {
	issueListCmd.Flags().StringSliceVarP(
		&issueLabels, "label", "l", []string{},
		"filter issues by label")
	issueListCmd.Flags().StringVarP(
		&issueState, "state", "s", "opened",
		"filter issues by state (all/opened/closed)")
	issueListCmd.Flags().StringVarP(
		&issueNumRet, "number", "n", "10",
		"number of issues to return")
	issueListCmd.Flags().BoolVarP(
		&issueAll, "all", "a", false,
		"list all issues on the project")
	issueListCmd.Flags().StringVar(
		&issueMilestone, "milestone", "",
		"filter issues by milestone/any/none")
	issueListCmd.Flags().StringVar(
		&issueAssignee, "assignee", "",
		"filter issues by assignee/any/none")
	issueListCmd.Flags().StringVar(
		&issueAuthor, "author", "",
		"filter issues by author")
	issueListCmd.Flags().BoolVarP(
		&issueExactMatch, "exact-match", "x", false,
		"match on the exact (case-insensitive) search terms")
	issueListCmd.Flags().StringVar(&issueOrder, "order", "updated_at", "display order (updated_at/created_at)")
	issueListCmd.Flags().StringVar(&issueSortedBy, "sort", "desc", "sort order (desc/asc)")

	issueCmd.AddCommand(issueListCmd)
	carapace.Gen(issueListCmd).FlagCompletion(carapace.ActionMap{
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
		"state": carapace.ActionValues("all", "opened", "closed"),
	})
	carapace.Gen(issueListCmd).PositionalCompletion(
		action.Remotes(),
	)
}
