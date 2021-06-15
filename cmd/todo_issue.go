package cmd

import (
	"strconv"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

var todoIssueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Add a Issue to Todo list",
	Example: heredoc.Doc(`
		lab todo issue 5678       #adds Issue 1234 to user's Todo list
	`),
	Hidden:           true,
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
	// https://github.com/xanzy/go-gitlab/pull/1130
	log.Fatal("Adding issues not implemented.")
}

func init() {
	todoCmd.AddCommand(todoIssueCmd)
}
