package cmd

import (
	"github.com/spf13/cobra"
)

// ciCmd represents the ci command
var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "Work with the GitLab CI pipeline for your refs",
	Long:  ``,
}

func init() {
	RootCmd.AddCommand(ciCmd)
}
