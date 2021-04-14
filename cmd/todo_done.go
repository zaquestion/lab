package cmd

import (
	"fmt"
	"log"
	"strconv"

	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var todoDoneCmd = &cobra.Command{
	Use:              "done",
	Short:            "Mark todo list entry as Done",
	Long:             ``,
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		toDoNum, err := strconv.Atoi(args[0])
		if err != nil {
			log.Fatal(err)
		}
		lab.TodoMarkDone(toDoNum)
		fmt.Println(toDoNum, "marked as Done")
	},
}

func init() {
	todoCmd.AddCommand(todoDoneCmd)
}
