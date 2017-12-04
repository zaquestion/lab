package cmd

import (
	"fmt"
	"log"

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
		remote, page, err := parseArgsRemote(args)
		if err != nil {
			log.Fatal(err)
		}
		if remote == "" {
			remote = forkedFromRemote
		}
		listOpts := gitlab.ListOptions{
			Page:    int(page),
			PerPage: 10,
		}

		// See if we're in a git repo or if global is set to determine
		// if this should be a personal snippet
		rn, _ := git.PathWithNameSpace(remote)
		if global || rn == "" {
			opts := gitlab.ListSnippetsOptions{
				ListOptions: listOpts,
			}
			snips, err := lab.SnippetList(&opts)
			if err != nil {
				log.Fatal(err)
			}
			for _, snip := range snips {
				fmt.Printf("#%d %s\n", snip.ID, snip.Title)
			}
			return
		}

		project, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}
		opts := gitlab.ListProjectSnippetsOptions{
			ListOptions: listOpts,
		}
		snips, err := lab.ProjectSnippetList(project.ID, &opts)
		if err != nil {
			log.Fatal(err)
		}
		for _, snip := range snips {
			fmt.Printf("#%d %s\n", snip.ID, snip.Title)
		}
	},
}

func init() {
	snippetCmd.AddCommand(snippetListCmd)
}
