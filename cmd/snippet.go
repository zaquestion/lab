package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// snippetCmd represents the snippet command
var snippetCmd = &cobra.Command{
	Use:     "snippet",
	Aliases: []string{"snip"},
	Short:   snippetCreateCmd.Short,
	Long:    snippetCreateCmd.Long,
	Run: func(cmd *cobra.Command, args []string) {
		if list, _ := cmd.Flags().GetBool("list"); list {
			snippetListCmd.Run(cmd, args)
			return
		}

		if browse, _ := cmd.Flags().GetBool("browse"); browse {
			snippetBrowseCmd.Run(cmd, args)
			return
		}

		if deleteID, _ := cmd.Flags().GetString("delete"); deleteID != "" {
			// append the args here so that remote can stil be
			// properly passed in
			snippetDeleteCmd.Run(cmd, append(args, deleteID))
			return
		}

		if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
			snippetCreateCmd.Run(cmd, args)
			return
		}

		if len(args) > 0 || file != "" {
			snippetCreateCmd.Run(cmd, args)
			return
		}

		cmd.Help()
	},
}

var (
	global bool
)

func init() {
	snippetCmd.PersistentFlags().BoolVarP(&global, "global", "g", false, "create as a personal snippet")
	snippetCmd.Flags().BoolP("list", "l", false, "list snippets")
	snippetCmd.Flags().BoolP("browse", "b", false, "browse snippets")
	snippetCmd.Flags().StringP("delete", "d", "", "delete snippet with id")
	// Create flags added in snippetCreate.go
	RootCmd.AddCommand(snippetCmd)
}
