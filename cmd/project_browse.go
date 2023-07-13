package cmd

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	projectFile    string
	projectBranch  string
	projectFileRef string
)

func projectBrowseGetPath(webURL string, defaultBranch string, branch string, file string, ref string) (path string) {
	if branch == "" {
		branch = defaultBranch
	}

	if ref != "" {
		path = webURL + "/-/tree/" + ref
		return path
	}

	if file != "" {
		path = webURL + "/-/blob/" + branch + "/" + file
	} else {
		path = webURL + "/-/tree/" + branch
	}

	return path
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

		err = browse(projectBrowseGetPath(p.WebURL, p.DefaultBranch, projectBranch, projectFile, projectFileRef))
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	projectBrowseCmd.Flags().StringVar(&projectFile, "file", "", "path to specified file")
	projectBrowseCmd.Flags().StringVar(&projectFileRef, "ref", "", "commit reference for file")
	projectBrowseCmd.Flags().StringVar(&projectBranch, "branch", "", "specific branch (overrides default branch)")
	projectCmd.AddCommand(projectBrowseCmd)
}
