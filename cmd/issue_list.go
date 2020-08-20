package cmd

import (
	"fmt"
	"log"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	issueLabels []string
	issueState  string
	issueSearch string
	issueNumRet int
	issueAll    bool
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
	Run: func(cmd *cobra.Command, args []string) {
		issues, err := issueList(args)
		if err != nil {
			log.Fatal(err)
		}
		for _, issue := range issues {
			fmt.Printf("#%d %s\n", issue.IID, issue.Title)
		}
	},
}

func issueList(args []string) ([]*gitlab.Issue, error) {
	rn, issueSearch, err := parseArgsRemoteString(args)
	if err != nil {
		return nil, err
	}

	opts := gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: issueNumRet,
		},
		Labels:  issueLabels,
		State:   &issueState,
		OrderBy: gitlab.String("updated_at"),
	}

	if issueSearch != "" {
		opts.Search = &issueSearch
	}

	num := issueNumRet
	if issueAll {
		num = -1
	}
	return lab.IssueList(rn, opts, num)
}

func init() {
	issueListCmd.Flags().StringSliceVarP(
		&issueLabels, "label", "l", []string{},
		"Filter issues by label")
	issueListCmd.Flags().StringVarP(
		&issueState, "state", "s", "opened",
		"Filter issues by state (opened/closed)")
	issueListCmd.Flags().IntVarP(
		&issueNumRet, "number", "n", 10,
		"Number of issues to return")
	issueListCmd.Flags().BoolVarP(
		&issueAll, "all", "a", false,
		"List all issues on the project")

	issueCmd.AddCommand(issueListCmd)
	carapace.Gen(issueListCmd).FlagCompletion(carapace.ActionMap{
		"state": carapace.ActionValues("opened", "closed"),
	})
	carapace.Gen(issueListCmd).PositionalCompletion(
		action.Remotes(),
	)
}
