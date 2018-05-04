package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
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

		printIssue(issue, rn)
	},
}

func printIssue(issue *gitlab.Issue, project string) {
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

func init() {
	issueCmd.AddCommand(issueShowCmd)
}
