package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
	zsh "github.com/rsteube/cobra-zsh-gen"
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
		rn, issueSearch, err := parseArgsRemoteString(args)
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

		if issueSearch != "" {
			opts.Search = &issueSearch
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

	zsh.Wrap(issueListCmd).MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	issueListCmd.MarkFlagCustom("state", "(opened closed)")
	issueCmd.AddCommand(issueListCmd)
}
