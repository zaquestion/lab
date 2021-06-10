package cmd

import (
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [shell]",
	Short: "Generates autocompletion for different shell implementations",
	Long: heredoc.Doc(`
		Generates shell autocompletion scripts for different implementations.

		These scripts can be directly sourced, though using pre-generated
		versions is recommended to avoid shell startup delay.
	`),
	Example: heredoc.Doc(`
		bash       : source <(lab completion)
		elvish     : eval(lab completion|slurp)
		fish       : lab completion | source
		oil        : source <(lab completion)
		powershell : lab completion | Out-String | Invoke-Expression
		xonsh      : exec($(lab completion))
		zsh        : source <(lab completion)
	`),
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
