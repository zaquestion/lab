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
		if len(args) == 0 && file == "" {
			cmd.Help()
			return
		}
		snippetCreateCmd.Run(cmd, args)
	},
}

func init() {
	// flags added in snippetCreate.go
	RootCmd.AddCommand(snippetCmd)
}
