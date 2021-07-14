package cmd

import (
	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var projectBrowseCmd = &cobra.Command{
	Use:              "browse [remote]",
	Aliases:          []string{"b"},
	Short:            "View project in a browser",
	Example:          "lab mr browse origin",
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, _, err := parseArgsRemoteAndID(args)
		if err != nil {
			log.Fatal(err)
		}

		p, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}

		err = browse(p.WebURL)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	projectCmd.AddCommand(projectBrowseCmd)
}
