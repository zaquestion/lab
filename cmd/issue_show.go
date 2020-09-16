package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	"github.com/zaquestion/lab/internal/config"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	issueShowConfig *viper.Viper
	issueShowPrefix string = "issue_show."
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

		issueShowConfig = config.LoadConfig("", "")

		noMarkdown, _ := cmd.Flags().GetBool("no-markdown")
		if err != nil {
			log.Fatal(err)
		}
		renderMarkdown := !noMarkdown

		printIssue(issue, rn, renderMarkdown)

		showComments, _ := cmd.Flags().GetBool("comments")
		if showComments == false {
			showComments = issueShowConfig.GetBool(issueShowPrefix + "comments")
		}
		if showComments {
			discussions, err := lab.IssueListDiscussions(rn, int(issueNum))
			if err != nil {
				log.Fatal(err)
			}

			since, err := cmd.Flags().GetString("since")
			if err != nil {
				log.Fatal(err)
			}

			printDiscussions(discussions, since, int(issueNum))
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

func printDiscussions(discussions []*gitlab.Discussion, since string, issueNum int) {
	NewAccessTime := time.Now().UTC()

	issueEntry := fmt.Sprintf("issue%d", issueNum)
	// if specified on command line use that, o/w use config, o/w Now
	var (
		CompareTime time.Time
		err         error
		sinceIsSet  = true
	)
	CompareTime, err = dateparse.ParseLocal(since)
	if err != nil || CompareTime.IsZero() {
		CompareTime = issueShowConfig.GetTime(issueShowPrefix + issueEntry)
		if CompareTime.IsZero() {
			CompareTime = time.Now().UTC()
		}
		sinceIsSet = false
	}

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
			if !time.Time(*note.CreatedAt).Equal(time.Time(*note.UpdatedAt)) {
				commented = "updated comment"
			}

			if !discussion.IndividualNote {
				indentNote = "    "

				if i == 0 {
					commented = "started a discussion"
				} else {
					indentHeader = "    "
				}
			}
			printit := color.New().PrintfFunc()
			printit(`
%s-----------------------------------`, indentHeader)

			if time.Time(*note.UpdatedAt).After(CompareTime) {
				printit = color.New(color.Bold).PrintfFunc()
			}
			printit(`
%s%s %s at %s

%s%s
`,
				indentHeader, note.Author.Username, commented, time.Time(*note.UpdatedAt).String(),
				indentNote, note.Body)
		}
	}

	if sinceIsSet == false {
		config.WriteConfigEntry(issueShowPrefix+issueEntry, NewAccessTime, "", "")
	}
}

func init() {
	issueShowCmd.Flags().BoolP("no-markdown", "M", false, "Don't use markdown renderer to print the issue description")
	issueShowCmd.Flags().BoolP("comments", "c", false, "Show comments for the issue")
	issueShowCmd.Flags().StringP("since", "s", "", "Show comments since specified date (format: 2020-08-21 14:57:46.808 +0000 UTC)")
	issueCmd.AddCommand(issueShowCmd)

	carapace.Gen(issueShowCmd).PositionalCompletion(
		action.Remotes(),
		action.Issues(issueList),
	)
}
