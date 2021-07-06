package cmd

import (
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"
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
	Example:          heredoc.Doc(`
		lab todo done
		lab todo done -a
		lab todo done -n 10
		lab todo done -p 
		lab todo done -t mr`),
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
		if len(args) == 0 {
			log.Fatalf("Specify todo id to be marked as done")
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
