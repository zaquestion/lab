package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var issueShowCmd = &cobra.Command{
	Use:        "show [remote] <id>",
	Aliases:    []string{"get"},
	ArgAliases: []string{"s"},
	Short:      "Describe an issue",
	Long:       ``,
	Run: func(cmd *cobra.Command, args []string) {
		rn, issueNum, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		issue, err := lab.IssueGet(rn, int(issueNum))
		if err != nil {
			log.Fatal(err)
		}

		noMarkdown, _ := cmd.Flags().GetBool("no-markdown")
		if err != nil {
			log.Fatal(err)
		}
		renderMarkdown := !noMarkdown

		printIssue(issue, rn, renderMarkdown)

		showComments, _ := cmd.Flags().GetBool("comments")
		if showComments {
			discussions, err := lab.IssueListDiscussions(rn, int(issueNum))
			if err != nil {
				log.Fatal(err)
			}

			printDiscussions(discussions)
		}
	},
}

func printIssue(issue *gitlab.Issue, project string, renderMarkdown bool) {
	milestone := "None"
	timestats := "None"
	dueDate := "None"
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
		r, _ := glamour.NewTermRenderer(
			glamour.WithStandardStyle("auto"),
		)

		issue.Description, _ = r.Render(issue.Description)
	}

	fmt.Printf(`
#%d %s
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
WebURL: %s
`,
		issue.IID, issue.Title, issue.Description, project, state, strings.Join(assignees, ", "),
		issue.Author.Username, milestone, dueDate, timestats,
		strings.Join(issue.Labels, ", "), issue.WebURL,
	)
}

func printDiscussions(discussions []*gitlab.Discussion) {
	// for available fields, see
	// https://godoc.org/github.com/xanzy/go-gitlab#Note
	// https://godoc.org/github.com/xanzy/go-gitlab#Discussion
	for _, discussion := range discussions {
		for i, note := range discussion.Notes {

			// skip system notes
			if note.System {
				continue
			}

			indentHeader, indentNote := "", ""
			commented := "commented"

			if !discussion.IndividualNote {
				indentNote = "    "

				if i == 0 {
					commented = "started a discussion"
				} else {
					indentHeader = "    "
				}
			}

			fmt.Printf(`
%s-----------------------------------
%s%s %s at %s

%s%s
`,
				indentHeader,
				indentHeader, note.Author.Username, commented, time.Time(*note.CreatedAt).String(),
				indentNote, note.Body)
		}
	}
}

func init() {
	issueShowCmd.Flags().BoolP("no-markdown", "M", false, "Don't use markdown renderer to print the issue description")
	issueShowCmd.Flags().BoolP("comments", "c", false, "Show comments for the issue")
	issueCmd.AddCommand(issueShowCmd)

	carapace.Gen(issueShowCmd).PositionalCompletion(
		action.Remotes(),
		action.Issues(issueList),
	)
}
