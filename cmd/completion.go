package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "generate shell completion files",
	Long:  ``,
}

var bashCompletionCmd = &cobra.Command{
	Use:   "bash",
	Short: "generate bash completion file",
	Long: `To load completion run

. <(lab completion bash)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
. <(lab completion bash)
`,
	Run: func(cmd *cobra.Command, args []string) {
		RootCmd.GenBashCompletion(os.Stdout)
	},
}

var zshCompletionCmd = &cobra.Command{
	Use:   "zsh",
	Short: "generate zsh completion file",
	Long:  `The author of lab has no idea how to load the generated zsh completion files, if you know how to load this output please open an issue at https://github.com/zaquestion/lab/issues and explain`,
	Run: func(cmd *cobra.Command, args []string) {
		RootCmd.GenZshCompletion(os.Stdout)
	},
}

func init() {
	completionCmd.AddCommand(bashCompletionCmd)
	completionCmd.AddCommand(zshCompletionCmd)
	RootCmd.AddCommand(completionCmd)
}
