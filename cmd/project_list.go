package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var projectListConfig struct {
	All        bool
	Owned      bool
	Membership bool
	Starred    bool

	Number int
}

var projectListCmd = &cobra.Command{
	Use:     "list [search]",
	Aliases: []string{"ls", "search"},
	Short:   "List your projects",
	Run: func(cmd *cobra.Command, args []string) {
		search, _, err := parseArgsStr(args)
		if err != nil {
			log.Fatal(err)
		}
		opt := gitlab.ListProjectsOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: projectListConfig.Number,
			},
			Simple:     gitlab.Bool(true),
			OrderBy:    gitlab.String("id"),
			Sort:       gitlab.String("asc"),
			Owned:      gitlab.Bool(projectListConfig.Owned),
			Membership: gitlab.Bool(projectListConfig.Membership),
			Starred:    gitlab.Bool(projectListConfig.Starred),
			Search:     gitlab.String(search),
		}
		num := projectListConfig.Number
		if projectListConfig.All {
			num = -1
		}
		projects, err := lab.ProjectList(opt, num)
		if err != nil {
			log.Fatal(err)
		}
		for _, p := range projects {
			fmt.Println(p.PathWithNamespace)
		}
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd)
	projectListCmd.Flags().BoolVarP(&projectListConfig.All, "all", "a", false, "List all projects on the instance")
	projectListCmd.Flags().BoolVarP(&projectListConfig.Owned, "mine", "m", false, "limit by your projects")
	projectListCmd.Flags().BoolVar(&projectListConfig.Membership, "member", false, "limit by projects which you are a member")
	projectListCmd.Flags().BoolVar(&projectListConfig.Starred, "starred", false, "limit by your starred projects")
	projectListCmd.Flags().IntVarP(&projectListConfig.Number, "number", "n", 100, "Number of projects to return")
}
