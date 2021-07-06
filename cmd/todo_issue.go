package cmd

import (
	"fmt"
	lab "github.com/zaquestion/lab/internal/gitlab"
	"strconv"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

var todoIssueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Add a Issue to Todo list",
	Example: heredoc.Doc(`
		lab todo issue 5678`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, err := getRemoteName("")
		if err != nil {
			log.Fatal(err)
		}

		num, err := strconv.Atoi(args[0])
		if err != nil {
			log.Fatal(err)
		}

		todoAddIssue(rn, num)

	},
}

func todoAddIssue(project string, issueNum int) {
	todoID, err := lab.TodoIssueCreate(project, issueNum)
	if err != nil {
		if err == lab.ErrNotModified {
			log.Fatalf("Todo entry already exists for Issue !%d", issueNum)
		}
		log.Fatal(err)
	}

	issue, err := lab.IssueGet(project, issueNum)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(todoID, issue.WebURL)
}

func init() {
	todoCmd.AddCommand(todoIssueCmd)
}
