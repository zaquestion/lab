package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/charmbracelet/glamour"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var issueShowCmd = &cobra.Command{
	Use:              "show [remote] <id>",
	Aliases:          []string{"get"},
	ArgAliases:       []string{"s"},
	Short:            "Describe an issue",
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {

		rn, issueNum, err := parseArgsRemoteAndID(args)
		if err != nil {
			log.Fatal(err)
		}

		issue, err := lab.IssueGet(rn, int(issueNum))
		if err != nil {
			log.Fatal(err)
		}

		renderMarkdown := false
		if isOutputTerminal() {
			noMarkdown, _ := cmd.Flags().GetBool("no-markdown")
			if err != nil {
				log.Fatal(err)
			}
			renderMarkdown = !noMarkdown
		}

		pager := newPager(cmd.Flags())
		defer pager.Close()

		printIssue(issue, rn, renderMarkdown)

		showComments, _ := cmd.Flags().GetBool("comments")
		if showComments {
			discussions, err := lab.IssueListDiscussions(rn, int(issueNum))
			if err != nil {
				log.Fatal(err)
			}

			since, err := cmd.Flags().GetString("since")
			if err != nil {
				log.Fatal(err)
			}

			printDiscussions(discussions, since, "issues", int(issueNum), renderMarkdown)
		}
	},
}

func printIssue(issue *gitlab.Issue, project string, renderMarkdown bool) {
	milestone := "None"
	timestats := "None"
	dueDate := "None"
	subscribed := "No"
	state := map[string]string{
		"opened": "Open",
		"closed": "Closed",
	}[issue.State]
	if issue.Milestone != nil {
		milestone = issue.Milestone.Title
	}
	if issue.TimeStats != nil && issue.TimeStats.HumanTimeEstimate != "" &&
		issue.TimeStats.HumanTotalTimeSpent != "" {
		timestats = fmt.Sprintf(
			"Estimated %s, Spent %s",
			issue.TimeStats.HumanTimeEstimate,
			issue.TimeStats.HumanTotalTimeSpent)
	}
	if issue.DueDate != nil {
		dueDate = time.Time(*issue.DueDate).String()
	}
	assignees := make([]string, len(issue.Assignees))
	if len(issue.Assignees) > 0 && issue.Assignees[0].Username != "" {
		for i, a := range issue.Assignees {
			assignees[i] = a.Username
		}
	}

	if renderMarkdown {
		r, err := getTermRenderer(glamour.WithAutoStyle())
		if err != nil {
			log.Fatal(err)
		}
		issue.Description, _ = r.Render(issue.Description)
	}

	relatedMRs, err := lab.ListMRsRelatedToIssue(project, issue.IID)
	if err != nil {
		log.Fatal(err)
	}
	closingMRs, err := lab.ListMRsClosingIssue(project, issue.IID)
	if err != nil {
		log.Fatal(err)
	}

	if issue.Subscribed {
		subscribed = "Yes"
	}

	fmt.Printf(
		heredoc.Doc(`#%d %s
			===================================
			%s
			-----------------------------------
			Project: %s
			Status: %s
			Assignees: %s
			Author: %s
			Milestone: %s
			Due Date: %s
			Time Stats: %s
			Labels: %s
			Related MRs: %s
			MRs that will close this Issue: %s
			Subscribed: %s
			WebURL: %s
		`),
		issue.IID, issue.Title, issue.Description, project, state, strings.Join(assignees, ", "),
		issue.Author.Username, milestone, dueDate, timestats,
		strings.Join(issue.Labels, ", "),
		strings.Trim(strings.Replace(fmt.Sprint(relatedMRs), " ", ",", -1), "[]"),
		strings.Trim(strings.Replace(fmt.Sprint(closingMRs), " ", ",", -1), "[]"),
		subscribed, issue.WebURL,
	)
}

func init() {
	issueShowCmd.Flags().BoolP("no-markdown", "M", false, "don't use markdown renderer to print the issue description")
	issueShowCmd.Flags().BoolP("comments", "c", false, "show comments for the issue")
	issueShowCmd.Flags().StringP("since", "s", "", "show comments since specified date (format: 2020-08-21 14:57:46.808 +0000 UTC)")
	issueCmd.AddCommand(issueShowCmd)

	carapace.Gen(issueShowCmd).PositionalCompletion(
		action.Remotes(),
		action.Issues(issueList),
	)
}
