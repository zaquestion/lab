package cmd

import (
	"fmt"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [shell]",
	Short: "Generates the shell autocompletion [bash, elvish, fish, powershell, xonsh, zsh]",
	Long: `Generates the shell autocompletion [bash, elvish, fish, powershell, xonsh, zsh]

Scripts can be directly sourced (though using pre-generated versions is recommended to avoid shell startup delay):
  bash       : source <(lab completion)
  elvish     : eval(lab completion|slurp)
  fish       : lab completion fish | source
  powershell : lab completion | Out-String | Invoke-Expression
  xonsh      : exec($(lab completion))
  zsh        : source <(lab completion)`,
	ValidArgs: []string{"bash", "elvish", "fish", "powershell", "xonsh", "zsh"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			fmt.Println(carapace.Gen(cmd).Snippet(args[0], true))
		} else {
			fmt.Println(carapace.Gen(cmd).Snippet("", true))
		}
	},
}

func init() {
	RootCmd.AddCommand(completionCmd)
}
