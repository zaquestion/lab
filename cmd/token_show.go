package cmd

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var tokenShowCmd = &cobra.Command{
	Use:     "show",
	Aliases: []string{"details", "current"},
	Short:   "show details about current Personal Access Token",
	Args:    cobra.MaximumNArgs(1),
	Example: heredoc.Doc(`
		lab token show
		lab token details
		lab token current`),

	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		tokendata, err := lab.GetCurrentPAT()
		if err != nil {
			log.Fatal(err)
		}
		dumpToken(tokendata)
	},
}

func init() {
	tokenCmd.AddCommand(tokenShowCmd)
}
