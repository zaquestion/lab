package cmd

import (
	"github.com/spf13/cobra"
)

// Hold --follow flag value that is common for all ci command
var followBridge bool

// ciCmd represents the ci command
var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "Work with GitLab CI pipelines and jobs",
	Long:  ``,
}

func init() {
	ciCmd.PersistentFlags().Bool("follow", false, "Follow downstream pipelines in a multi-projects setup")
	RootCmd.AddCommand(ciCmd)
}
