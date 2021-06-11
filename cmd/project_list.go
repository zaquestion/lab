package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var projectListConfig struct {
	All        bool
	Owned      bool
	Membership bool
	Starred    bool
	Number     string
}

var projectListCmd = &cobra.Command{
	Use:              "list [search]",
	Aliases:          []string{"ls", "search"},
	Short:            "List your projects",
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		search, _, err := parseArgsStringAndID(args)
		if err != nil {
			log.Fatal(err)
		}

		num, err := strconv.Atoi(projectListConfig.Number)
		if projectListConfig.All || (err != nil) {
			num = -1
		}

		opt := gitlab.ListProjectsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: num,
			},
			Simple:     gitlab.Bool(true),
			OrderBy:    gitlab.String("id"),
			Sort:       gitlab.String("asc"),
			Owned:      gitlab.Bool(projectListConfig.Owned),
			Membership: gitlab.Bool(projectListConfig.Membership),
			Starred:    gitlab.Bool(projectListConfig.Starred),
			Search:     gitlab.String(search),
		}
		projects, err := lab.ProjectList(opt, num)
		if err != nil {
			log.Fatal(err)
		}

		pager := newPager(cmd.Flags())
		defer pager.Close()

		for _, p := range projects {
			fmt.Println(p.PathWithNamespace)
		}
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd)
	projectListCmd.Flags().BoolVarP(&projectListConfig.All, "all", "a", false, "list all projects on the instance")
	projectListCmd.Flags().BoolVarP(&projectListConfig.Owned, "mine", "m", false, "limit by your projects")
	projectListCmd.Flags().BoolVar(&projectListConfig.Membership, "member", false, "limit by projects which you are a member")
	projectListCmd.Flags().BoolVar(&projectListConfig.Starred, "starred", false, "limit by your starred projects")
	projectListCmd.Flags().StringVarP(&projectListConfig.Number, "number", "n", "100", "Number of projects to return")
	projectListCmd.Flags().SortFlags = false
}
