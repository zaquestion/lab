package cmd

import (
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var regListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List your registries",
	Example: heredoc.Doc(`
		lab reg list`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, _, err := parseArgsRemoteAndProject(args)
		if err != nil {
			log.Fatal(err)
		}

		num, err := strconv.Atoi(projectListConfig.Number)
		if projectListConfig.All || (err != nil) {
			num = -1
		}

		project, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
			return
		}

		opt := gitlab.ListRegistryRepositoriesOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: num,
			},
			Tags:      gitlab.Bool(true),
			TagsCount: gitlab.Bool(true),
		}
		registries, err := lab.ContainerRegistryList(project.ID, &opt, 0)
		if err != nil {
			log.Fatal(err)
			return
		}

		for _, r := range registries {
			fmt.Printf("!%d %s\n", r.ID, r.Path)
		}
	},
}

func init() {
	regCmd.AddCommand(regListCmd)
}
