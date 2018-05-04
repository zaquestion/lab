package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	issueLabels []string
	issueState  string
)

var issueListCmd = &cobra.Command{
	Use:     "list [remote] [page]",
	Aliases: []string{"ls"},
	Short:   "List issues",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		rn, page, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		issues, err := lab.IssueList(rn, &gitlab.ListProjectIssuesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    int(page),
				PerPage: 10,
			},
			Labels:  issueLabels,
			State:   &issueState,
			OrderBy: gitlab.String("updated_at"),
		})
		if err != nil {
			log.Fatal(err)
		}
		for _, issue := range issues {
			fmt.Printf("#%d %s\n", issue.IID, issue.Title)
		}
	},
}

func init() {
	issueListCmd.Flags().StringSliceVarP(
		&issueLabels, "label", "l", []string{}, "filter issues by label")
	issueListCmd.Flags().StringVarP(
		&issueState, "state", "s", "opened",
		"filter issues by state (opened/closed)")
	issueCmd.AddCommand(issueListCmd)
}
