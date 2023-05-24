package cmd

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var tokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "list details about all Personal Access Tokens",
	Args:  cobra.MaximumNArgs(1),
	Example: heredoc.Doc(`
		lab token list`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		tokens, err := lab.GetAllPATs()
		if err != nil {
			log.Fatal(err)
		}

		all, _ := cmd.Flags().GetBool("all")
		for _, token := range tokens {
			if token.Active || all {
				dumpToken(token)
			}
		}
	},
}

func init() {
	tokenListCmd.Flags().BoolP("all", "a", false, "list all tokens (including inactive)")
	tokenCmd.AddCommand(tokenListCmd)
}
