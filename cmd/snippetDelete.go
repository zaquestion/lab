package cmd

import (
	"fmt"
	"log"
	"strconv"

	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// snippetDeleteCmd represents the snippetDelete command
var snippetDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a personal snippet by ID",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id, err := strconv.ParseInt(args[0], 0, 64)
		if err != nil {
			log.Fatal(err)
		}
		err = lab.SnippetDelete(int(id))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Snippet #%d deleted\n", id)
	},
}

func init() {
	snippetCmd.AddCommand(snippetDeleteCmd)
}
