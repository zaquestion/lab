package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

// mrCmd represents the mr command
var mergeRequestCmd = &cobra.Command{
	Use:   "merge-request [remote [branch]]",
	Short: mrCreateCmd.Short,
	Long:  mrCreateCmd.Long,
	Args:  mrCreateCmd.Args,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[WARN] `lab merge-request` will be deprecated by `lab mr create` before v1.0 ")
		runMRCreate(cmd, args)
	},
}

func init() {
	RootCmd.AddCommand(mergeRequestCmd)
}
