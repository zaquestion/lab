package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// snippetDeleteCmd represents the snippetDelete command
var snippetDeleteCmd = &cobra.Command{
	Use:   "delete [remote] <id>",
	Short: "Delete a project or personal snippet",
	Long:  ``,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}
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
