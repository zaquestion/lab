package cmd

import (
	"fmt"
	"log"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var projectListCmd = &cobra.Command{
	Use:     "list [page]",
	Aliases: []string{"ls"},
	Short:   "List all projects",
	Run: func(cmd *cobra.Command, args []string) {
		page := 0
		if len(args) == 1 {
			n, err := strconv.ParseInt(args[0], 0, 64)
			if err != nil {
				log.Fatal(errors.Errorf("%s is not a valid page number", args[0]))
			}
			page = int(n)
		}
		projects, err := lab.ProjectList(&gitlab.ListProjectsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: 10,
			},
			Simple:  gitlab.Bool(true),
			OrderBy: gitlab.String("id"),
			Sort:    gitlab.String("asc"),
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
}
