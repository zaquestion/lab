package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var projectListCmd = &cobra.Command{
	Use:     "list [search [page]]",
	Aliases: []string{"ls", "search"},
	Short:   "List your projects",
	Run: func(cmd *cobra.Command, args []string) {
		search, page, err := parseArgsStr(args)
		if err != nil {
			log.Fatal(err)
		}
		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			log.Fatal(err)
		}
		projects, err := lab.ProjectList(&gitlab.ListProjectsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    int(page),
				PerPage: 10,
			},
			Simple:  gitlab.Bool(true),
			OrderBy: gitlab.String("id"),
			Sort:    gitlab.String("asc"),
			Owned:   gitlab.Bool(!all),
			Search:  gitlab.String(search),
		})
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
	projectListCmd.Flags().BoolP("all", "a", false, "list all projects")
}
