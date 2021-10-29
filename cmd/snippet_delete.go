package cmd

import (
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// snippetDeleteCmd represents the snippetDelete command
var snippetDeleteCmd = &cobra.Command{
	Use:   "delete [remote] <id>",
	Short: "Delete a project or personal snippet",
	Example: heredoc.Doc(`
		lab snippet delete origin 2147103
		lab snippet delete -g 2147104`),
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsRemoteAndID(args)
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

		err = lab.ProjectSnippetDelete(rn, int(id))
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	snippetCmd.AddCommand(snippetDeleteCmd)
	carapace.Gen(snippetDeleteCmd).PositionalCompletion(
		action.Remotes(),
		action.Snippets(snippetList),
	)
}
