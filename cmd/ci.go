package cmd

import (
	"github.com/spf13/cobra"
)

// Hold --follow and --bridge values that are common to all ci command
var (
	followBridge bool
	bridgeName   string
)

// ciCmd represents the ci command
var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "Work with GitLab CI pipelines and jobs",
}

func init() {
	ciCmd.PersistentFlags().Bool("follow", false, "Follow bridge jobs (downstream pipelines) in a multi-projects setup")
	ciCmd.PersistentFlags().String("bridge", "", "Bridge job (downstream pipeline) name")
	RootCmd.AddCommand(ciCmd)
}
