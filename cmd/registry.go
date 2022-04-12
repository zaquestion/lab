package cmd

import (
	"github.com/spf13/cobra"
)

// regCmd represents the reg command
var regCmd = &cobra.Command{
	Use:              "registry",
	Aliases:          []string{"reg"},
	Short:            `Describe and list container registries and tags`,
	Long:             ``,
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 || len(args) > 2 {
			cmd.Help()
			return
		}

		regListCmd.Run(cmd, args)
	},
}

func init() {
	RootCmd.AddCommand(regCmd)
}
