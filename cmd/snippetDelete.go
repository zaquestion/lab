package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// snippetDeleteCmd represents the snippetDelete command
var snippetDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a snippet by ID",
	Long:  ``,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		remote, id, err := parseArgsRemote(args)
		if err != nil {
			log.Fatal(err)
		}
		if remote == "" {
			remote = forkedFromRemote
		}
		rn, _ := git.PathWithNameSpace(remote)
		if global || rn == "" {
			err = lab.SnippetDelete(int(id))
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Snippet #%d deleted\n", id)
			return
		}

		project, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}
		err = lab.ProjectSnippetDelete(project.ID, int(id))
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	snippetCmd.AddCommand(snippetDeleteCmd)
}
