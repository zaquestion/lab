package cmd

import (
	"fmt"
	"log"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var issueListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List issues",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		rn, err := git.PathWithNameSpace("origin")
		if err != nil {
			log.Fatal(err)
		}
		page := 0
		if len(args) == 1 {
			page, err = strconv.Atoi(args[0])
			if err != nil {
				log.Fatal(err)
			}
		}

		issues, err := lab.IssueList(rn, &gitlab.ListProjectIssuesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: 10,
			},
			State:   gitlab.String("opened"),
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
	issueCmd.AddCommand(issueListCmd)
}
