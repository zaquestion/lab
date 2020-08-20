package cmd

import (
	"fmt"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:       "completion",
	Short:     "Generates the shell autocompletion [bash, elvish, fish, powershell, zsh]",
    Long: `Generates the shell autocompletion [bash, elvish, fish, powershell, zsh]

Most scripts can be direcly sourced (though using pre-generated versions is recommended):
  bash       : source <(lab completion bash)
  elvish     : lab completion elvish > lab.elv; -source lab.elv
  fish       : lab completion fish | source
  powershell : lab completion powershell | Out-String | Invoke-Expression
  zsh        : source <(lab completion zsh)`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "elvish", "fish", "powershell", "zsh"},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(carapace.Gen(cmd).Snippet(args[0]))
	},
}

func init() {
	RootCmd.AddCommand(completionCmd)
}
