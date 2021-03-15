package cmd

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	todoType   string
	todoNumRet string
	targetType string
)

var todoListCmd = &cobra.Command{
	Use:              "list",
	Aliases:          []string{"ls"},
	Short:            "List todos",
	Long:             ``,
	Example:          `lab todo list                        # list open todos"`,
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		todos, err := todoList(args)
		if err != nil {
			log.Fatal(err)
		}

		pager := NewPager(cmd.Flags())
		defer pager.Close()

		for _, todo := range todos {
			fmt.Printf("%d %s\n", todo.ID, todo.TargetURL)
		}
	},
}

func todoList(args []string) ([]*gitlab.Todo, error) {
	num, err := strconv.Atoi(todoNumRet)
	if err != nil {
		num = -1
	}

	opts := gitlab.ListTodosOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: num,
		},
	}

	var lstr = strings.ToLower(todoType)
	switch {
	case lstr == "mr":
		targetType = "MergeRequest"
		opts.Type = &targetType
	case lstr == "issue":
		targetType = "Issue"
		opts.Type = &targetType
	}

	return lab.TodoList(opts, num)
}

func init() {
	todoListCmd.Flags().StringVarP(
		&todoType, "type", "t", "all",
		"filter todos by type (all/mr/issue)")
	todoListCmd.Flags().StringVarP(
		&todoNumRet, "number", "n", "10",
		"number of todos to return")

	todoCmd.AddCommand(todoListCmd)
}
