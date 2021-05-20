package cmd

import (
	"fmt"
	"os"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [shell]",
	Short: "Generates the shell autocompletion [bash, elvish, fish, oil, powershell, xonsh, zsh]",
	Long: `Generates the shell autocompletion [bash, elvish, fish, oil, powershell, xonsh, zsh]

Scripts can be directly sourced (though using pre-generated versions is recommended to avoid shell startup delay):
  bash       : source <(lab completion)
  elvish     : eval(lab completion|slurp)
  fish       : lab completion | source
  oil        : source <(lab completion)
  powershell : lab completion | Out-String | Invoke-Expression
  xonsh      : exec($(lab completion))
  zsh        : source <(lab completion)`,
	ValidArgs: []string{"bash", "elvish", "fish", "oil", "powershell", "xonsh", "zsh"},
	Run: func(cmd *cobra.Command, args []string) {
		shell := ""
		if len(args) > 0 {
			shell = args[0]
		}
		if script, err := carapace.Gen(cmd).Snippet(shell); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		} else {
			fmt.Println(script)
		}
	},
}

func init() {
	RootCmd.AddCommand(completionCmd)
}
