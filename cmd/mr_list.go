package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrLabels []string
var mrState string
var mrPerPage int

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list [remote] [page]",
	Aliases: []string{"ls"},
	Short:   "List merge requests",
	Long:    ``,
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rn, page, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		mrs, err := lab.MRList(rn, &gitlab.ListProjectMergeRequestsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    int(page),
				PerPage: mrPerPage,
			},
			Labels:  mrLabels,
			State:   &mrState,
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
	listCmd.Flags().StringVarP(
		&mrState, "state", "s", "opened",
		"filter merge requests by state (opened/closed/merged)")
	listCmd.Flags().IntVarP(
		&mrPerPage, "per-page", "n", 10,
		"preset number of returned merge requests")
	mrCmd.AddCommand(listCmd)
}
