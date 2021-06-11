package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	all bool
)

var todoDoneCmd = &cobra.Command{
	Use:              "done",
	Short:            "Mark todo list entry as Done",
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		if all {
			err := lab.TodoMarkAllDone()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("All Todo entries marked as Done")
			return
		}

		toDoNum, err := strconv.Atoi(args[0])
		if err != nil {
			log.Fatal(err)
		}
		err = lab.TodoMarkDone(toDoNum)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(toDoNum, "marked as Done")
	},
}

func init() {
	todoDoneCmd.Flags().BoolVarP(&all, "all", "a", false, "mark all Todos as Done")
	todoCmd.AddCommand(todoDoneCmd)
}
