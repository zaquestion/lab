package cmd

import (
	"github.com/spf13/cobra"
)

// repoCmd represents the repo command
var projectCmd = &cobra.Command{
	Use:              "project",
	Aliases:          []string{"repo"},
	Short:            "Perform project level operations on GitLab",
	PersistentPreRun: labPersistentPreRun,
}

func init() {
	RootCmd.AddCommand(projectCmd)
}
