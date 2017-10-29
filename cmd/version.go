package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/git"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		git := git.New("version")
		git.Stdout = nil
		git.Stderr = nil
		version, _ := git.Output()
		fmt.Printf("%s%s\n", string(version), "lab version 0.5.0")
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
