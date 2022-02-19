package cmd

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var todoIssueCmd = &cobra.Command{
	Use:   "issue [remote] <id>",
	Short: "Add a Issue to Todo list",
	Example: heredoc.Doc(`
		lab todo issue 5678               #adds issue 5678 to user's Todo list
		lab todo issue otherRemote 91011`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, num, err := parseArgsRemoteAndID(args)
		if err != nil {
			log.Fatal(err)
		}

		todoAddIssue(rn, int(num))

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
