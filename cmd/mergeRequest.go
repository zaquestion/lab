package cmd

import (
	"github.com/spf13/cobra"
)

// mrCmd represents the mr command
var mergeRequestCmd = &cobra.Command{
	Use:   "merge-request",
	Short: "Open Merge Request on GitLab",
	Long:  `Currently only supports MRs into origin/master`,
	Args:  cobra.ExactArgs(0),
	Run:   runMRCreate,
}

func init() {
	RootCmd.AddCommand(mergeRequestCmd)
}
