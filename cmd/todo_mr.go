package cmd

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var todoMRCmd = &cobra.Command{
	Use:   "mr [remote] <id>",
	Short: "Add a Merge Request to Todo list",
	Example: heredoc.Doc(`
			lab todo mr 1234              #adds MR 1234 to user's Todo list
			lab todo mr otherRemote 5678`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, num, err := parseArgsRemoteAndID(args)
		if err != nil {
			log.Fatal(err)
		}

		todoAddMergeRequest(rn, int(num))
	},
}

func todoAddMergeRequest(remote string, mrNum int) {
	todoID, err := lab.TodoMRCreate(remote, mrNum)
	if err != nil {
		if err == lab.ErrNotModified {
			log.Fatalf("Todo entry already exists for MR !%d", mrNum)
		}
		log.Fatal(err)
	}

	mr, err := lab.MRGet(remote, mrNum)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(todoID, mr.WebURL)
}

func init() {
	todoCmd.AddCommand(todoMRCmd)
}
