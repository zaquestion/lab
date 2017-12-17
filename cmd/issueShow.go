package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var issueShowCmd = &cobra.Command{
	Use:     "show [remote]",
	Aliases: []string{"get", "s"},
	Short:   "describe an issue",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		remote, issueNum, err := parseArgsRemote(args)
		if err != nil {
			log.Fatal(err)
		}
		if remote == "" {
			remote = forkedFromRemote
		}
		rn, err := git.PathWithNameSpace(remote)
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
	assignee := "None"
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
	if issue.Assignee.Username != "" {
		assignee = issue.Assignee.Username
	}

	fmt.Printf(`
#%d %s
===================================
%s
-----------------------------------
Project: %s
Status: %s
Assignee: %s
Author: %s
Milestone: %s
Due Date: %s
Time Stats: %s
Labels: %s
WebURL: %s
`,
		issue.IID, issue.Title, issue.Description, project, state, assignee,
		issue.Author.Username, milestone, dueDate, timestats,
		strings.Join(issue.Labels, ", "), issue.WebURL,
	)
}

func init() {
	issueCmd.AddCommand(issueShowCmd)
}
