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
	issueLabels     []string
	issueMilestone  string
	issueState      string
	issueSearch     string
	issueNumRet     string
	issueAll        bool
	issueExactMatch bool
	issueAssignee   string
	issueAssigneeID *int
	issueAuthor     string
	issueAuthorID   *int
	issueOrder      string
	issueSortedBy   string
)

var issueListCmd = &cobra.Command{
	Use:     "list [remote] [search]",
	Aliases: []string{"ls", "search"},
	Short:   "List issues",
	Long:    ``,
	Example: `lab issue list                        # list all open issues
lab issue list "search terms"         # search issues for "search terms"
lab issue search "search terms"       # same as above
lab issue list remote "search terms"  # search "remote" for issues with "search terms"`,
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		issues, err := issueList(args)
		if err != nil {
			log.Fatal(err)
		}

		pager := NewPager(cmd.Flags())
		defer pager.Close()

		for _, issue := range issues {
			fmt.Printf("#%d %s\n", issue.IID, issue.Title)
		}
	},
}

func issueList(args []string) ([]*gitlab.Issue, error) {
	rn, issueSearch, err := parseArgsRemoteAndProject(args)
	if err != nil {
		return nil, err
	}

	labels, err := MapLabels(rn, issueLabels)
	if err != nil {
		return nil, err
	}

	if issueMilestone != "" {
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
			log.Fatal(fmt.Errorf("%s user not found\n", issueAuthor))
		}
	}

	if issueAssignee != "" {
		issueAssigneeID = getUserID(issueAssignee)
		if issueAssigneeID == nil {
			log.Fatal(fmt.Errorf("%s user not found\n", issueAssignee))
		}
	}

	orderBy := gitlab.String(issueOrder)

	sort := gitlab.String(issueSortedBy)

	opts := gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: num,
		},
		Labels:     labels,
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
		"filter issues by milestone")
	issueListCmd.Flags().StringVar(
		&issueAssignee, "assignee", "",
		"filter issues by assignee")
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
		"state": carapace.ActionValues("all", "opened", "closed"),
	})
	carapace.Gen(issueListCmd).PositionalCompletion(
		action.Remotes(),
	)
}
