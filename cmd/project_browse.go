package cmd

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	projectFile   string
	projectGitRef string
)

func projectBrowseGetPath(webURL string, defaultBranch string, ref string, file string) (path string) {
	if ref == "" {
		ref = defaultBranch
	}

	path = webURL + "/-/blob/" + ref

	if file != "" {
		path += "/" + file
	}

	return
}

var projectBrowseCmd = &cobra.Command{
	Use:     "browse [remote]",
	Aliases: []string{"b"},
	Short:   "View project in a browser",
	Example: heredoc.Doc(`
		lab project browse origin
		lab project browse --file arch/x86/Makefile
		lab project b --ref ce697ccee1`),
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

		err = browse(projectBrowseGetPath(p.WebURL, p.DefaultBranch, projectGitRef, projectFile))
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	projectBrowseCmd.Flags().StringVar(&projectFile, "file", "", "path to specified file")
	projectBrowseCmd.Flags().StringVar(&projectGitRef, "ref", "", "git reference (branch, tag, or SHA)")
	projectCmd.AddCommand(projectBrowseCmd)
}
