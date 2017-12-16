package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrLabels []string

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List merge requests",
	Long:    ``,
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		remote, page, err := parseArgsRemote(args)
		if err != nil {
			log.Fatal(err)
		}
		if remote == "" {
			remote = forkedFromRemote
		}
		rn, err := git.PathWithNameSpace(remote)
		if err != nil {
			log.Fatal(err)
		}

		mrs, err := lab.ListMRs(rn, &gitlab.ListProjectMergeRequestsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    int(page),
				PerPage: 10,
			},
			Labels:  mrLabels,
			State:   gitlab.String("opened"),
			OrderBy: gitlab.String("updated_at"),
		})
		if err != nil {
			log.Fatal(err)
		}
		for _, mr := range mrs {
			fmt.Printf("#%d %s\n", mr.IID, mr.Title)
		}
	},
}

func init() {
	listCmd.Flags().StringSliceVarP(
		&mrLabels, "label", "l", []string{}, "filter merge requests by label")
	mrCmd.AddCommand(listCmd)
}
