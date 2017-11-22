package cmd

import (
	"fmt"
	"log"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// snippetListCmd represents the snippetList command
var snippetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List personal or project snippets",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		page := 0
		if len(args) == 1 {
			var err error
			page, err = strconv.Atoi(args[0])
			if err != nil {
				log.Fatal(err)
			}
		}
		var snips []*gitlab.Snippet
		if rn, _ := git.PathWithNameSpace(forkRemote); rn != "" {
			project, err := lab.FindProject(rn)
			if err != nil {
				log.Fatal(err)
			}
			opts := gitlab.ListProjectSnippetsOptions{
				ListOptions: gitlab.ListOptions{
					Page:    page,
					PerPage: 10,
				},
			}
			snips, err = lab.ProjectSnippetList(project.ForkedFromProject.ID, &opts)
			if err != nil {
				log.Fatal(err)
			}
			// Try user fork if failed to create on forkedFromRepo.
			// Seemingly the next best bet
			if len(snips) == 0 {
				snips, err = lab.ProjectSnippetList(project.ID, &opts)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
		if len(snips) == 0 {
			opts := gitlab.ListSnippetsOptions{
				ListOptions: gitlab.ListOptions{
					Page:    page,
					PerPage: 10,
				},
			}
			var err error
			snips, err = lab.SnippetList(&opts)
			if err != nil {
				log.Fatal(err)
			}
		}
		for _, snip := range snips {
			fmt.Printf("#%d %s\n", snip.ID, snip.Title)
		}
	},
}

func init() {
	snippetCmd.AddCommand(snippetListCmd)
}
