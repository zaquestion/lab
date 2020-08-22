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

var mrShowCmd = &cobra.Command{
	Use:        "show [remote] <id>",
	Aliases:    []string{"get"},
	ArgAliases: []string{"s"},
	Short:      "Describe a merge request",
	Long:       ``,
	Run: func(cmd *cobra.Command, args []string) {
		rn, mrNum, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		mr, err := lab.MRGet(rn, int(mrNum))
		if err != nil {
			log.Fatal(err)
		}

		noMarkdown, _ := cmd.Flags().GetBool("no-markdown")
		if err != nil {
			log.Fatal(err)
		}
		renderMarkdown := !noMarkdown

		printMR(mr, rn, renderMarkdown)

		showComments, _ := cmd.Flags().GetBool("comments")
		if showComments {
			discussions, err := lab.MRListDiscussions(rn, int(mrNum))
			if err != nil {
				log.Fatal(err)
			}

			printMRDiscussions(discussions)
		}
	},
}

func printMR(mr *gitlab.MergeRequest, project string, renderMarkdown bool) {
	assignee := "None"
	milestone := "None"
	labels := "None"
	state := map[string]string{
		"opened": "Open",
		"closed": "Closed",
		"merged": "Merged",
	}[mr.State]

	if mr.Assignee != nil && mr.Assignee.Username != "" {
		assignee = mr.Assignee.Username
	}
	if mr.Milestone != nil {
		milestone = mr.Milestone.Title
	}
	if len(mr.Labels) > 0 {
		labels = strings.Join(mr.Labels, ", ")
	}

	if renderMarkdown {
		r, _ := glamour.NewTermRenderer(
			glamour.WithStandardStyle("auto"),
		)

		mr.Description, _ = r.Render(mr.Description)
	}

	fmt.Printf(`
#%d %s
===================================
%s
-----------------------------------
Project: %s
Branches: %s->%s
Status: %s
Assignee: %s
Author: %s
Milestone: %s
Labels: %s
WebURL: %s
`,
		mr.IID, mr.Title, mr.Description, project, mr.SourceBranch,
		mr.TargetBranch, state, assignee,
		mr.Author.Username, milestone, labels, mr.WebURL)
}

func printMRDiscussions(discussions []*gitlab.Discussion) {
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

			fmt.Printf(`
%s-----------------------------------
%s%s %s at %s

%s%s
`,
				indentHeader,
				indentHeader, note.Author.Username, commented, time.Time(*note.UpdatedAt).String(),
				indentNote, note.Body)
		}
	}
}

func init() {
	mrShowCmd.Flags().BoolP("no-markdown", "M", false, "Don't use markdown renderer to print the issue description")
	mrShowCmd.Flags().BoolP("comments", "c", false, "Show comments for the merge request")
	mrCmd.AddCommand(mrShowCmd)
	carapace.Gen(mrShowCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
