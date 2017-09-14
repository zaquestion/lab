package cmd

import (
	"github.com/spf13/cobra"
)

// mrCmd represents the mr command
var mrCmd = &cobra.Command{
	Use:   "mr",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	RootCmd.AddCommand(mrCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// mrCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// mrCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
