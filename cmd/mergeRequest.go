package cmd

import (
	"github.com/spf13/cobra"
)

// mrCmd represents the mr command
var mergeRequestCmd = &cobra.Command{
	Use:   "merge-request",
	Short: "Open a merge request on GitLab",
	Long:  `Currently only supports MRs into master`,
	Args:  cobra.ExactArgs(0),
	Run:   runMRCreate,
}

func init() {
	RootCmd.AddCommand(mergeRequestCmd)
}
