package cmd

import (
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

		if deleteID, _ := cmd.Flags().GetInt("delete"); deleteID != 0 {
			// append the args here so that remote can stil be
			// properly passed in
			snippetDeleteCmd.Run(cmd, append(args, string(deleteID)))
			return
		}
		if len(args) == 0 && file == "" {
			cmd.Help()
			return
		}
		snippetCreateCmd.Run(cmd, args)
	},
}

var (
	global bool
)

func init() {
	snippetCmd.PersistentFlags().BoolVarP(&global, "global", "g", false, "create as a personal snippet")
	snippetCmd.Flags().BoolP("list", "l", false, "list snippets")
	snippetCmd.Flags().IntP("delete", "d", 0, "delete snippet with id")
	// Create flags added in snippetCreate.go
	RootCmd.AddCommand(snippetCmd)
}
