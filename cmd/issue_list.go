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
	issueNumRet int
	issueAll    bool
)

var issueListCmd = &cobra.Command{
	Use:     "list [remote]",
	Aliases: []string{"ls"},
	Short:   "List issues",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		rn, _, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		opts := gitlab.ListProjectIssuesOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: issueNumRet,
			},
			Labels:  issueLabels,
			State:   &issueState,
			OrderBy: gitlab.String("updated_at"),
		}
		num := issueNumRet
		if issueAll {
			num = -1
		}
		issues, err := lab.IssueList(rn, opts, num)
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
		&issueLabels, "label", "l", []string{}, "Filter issues by label")
	issueListCmd.Flags().StringVarP(
		&issueState, "state", "s", "opened",
		"Filter issues by state (opened/closed)")
	issueListCmd.Flags().IntVarP(
		&issueNumRet, "number", "n", 10,
		"Number of issues to return")
	issueListCmd.Flags().BoolVarP(&issueAll, "all", "a", false, "List all issues on the project")
	issueCmd.AddCommand(issueListCmd)
}
