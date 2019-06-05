package cmd

import (
	"os"

	"github.com/spf13/cobra"
	zsh "github.com/rsteube/cobra-zsh-gen"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:       "completion",
	Short:     "Generates the shell autocompletion",
	Long:      `'completion bash' generates the bash and 'completion zsh' the zsh autocompletion`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh"},
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			RootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			// currently using BashCompletionFunction variable for additional custom completions in zsh
			RootCmd.BashCompletionFunction = zshCompletionFunction
			zsh.Wrap(RootCmd).GenZshCompletion(os.Stdout)
		default:
			println("only 'bash' or 'zsh' allowed")
		}
	},
}

func init() {
	RootCmd.AddCommand(completionCmd)
}
