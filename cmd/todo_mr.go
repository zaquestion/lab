package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var todoMRCmd = &cobra.Command{
	Use:              "mr",
	Short:            "Add a Merge Request to Todo list",
	Example:          "lab todo mr 1234    #adds MR 1234 to user's Todo list",
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, err := getRemoteName("")
		if err != nil {
			log.Fatal(err)
		}

		num, err := strconv.Atoi(args[0])
		if err != nil {
			log.Fatal(err)
		}

		todoAddMergeRequest(rn, num)
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
