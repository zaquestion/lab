package cmd

import (
	"log"
	"path"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/browser"
	"github.com/zaquestion/lab/internal/gitlab"
)

var browse = browser.Open

var issueBrowseCmd = &cobra.Command{
	Use:     "browse [remote] <id>",
	Aliases: []string{"b"},
	Short:   "View issue in a browser",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		rn, num, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		project, err := gitlab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}

		issueURL := path.Join(project.WebURL, "issues")
		if num > 0 {
			issueURL = path.Join(issueURL, strconv.FormatInt(num, 10))
		}

		err = browse(issueURL)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	issueBrowseCmd.MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	issueBrowseCmd.MarkZshCompPositionalArgumentCustom(2, "__lab_completion_issue $words[2]")
	issueCmd.AddCommand(issueBrowseCmd)
}
