package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var projectBrowseCmd = &cobra.Command{
	Use:     "browse [remote]",
	Aliases: []string{"b"},
	Short:   "View project in a browser",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		remote, _, err := parseArgsRemote(args)
		if err != nil {
			log.Fatal(err)
		}
		if remote == "" {
			remote = forkedFromRemote
		}

		name, err := git.PathWithNameSpace(remote)
		if err != nil {
			log.Fatal(err)
		}

		p, err := lab.FindProject(name)
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
