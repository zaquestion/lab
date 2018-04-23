package cmd

import (
	"github.com/spf13/cobra"
)

// ciCmd represents the ci command
var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "Work with GitLab CI pipelines and jobs",
	Long:  ``,
}

func init() {
	RootCmd.AddCommand(ciCmd)
}
