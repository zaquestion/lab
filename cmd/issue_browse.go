package cmd

import (
	"github.com/MakeNowJust/heredoc/v2"
	"strconv"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	"github.com/zaquestion/lab/internal/gitlab"
)

var issueBrowseCmd = &cobra.Command{
	Use:     "browse [remote] <id>",
	Aliases: []string{"b"},
	Short:   "View issue in a browser",
	Example: heredoc.Doc(`
		lab issue browse
		lab issue browse upstream 22`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, num, err := parseArgsRemoteAndID(args)
		if err != nil {
			log.Fatal(err)
		}

		project, err := gitlab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}

		// path.Join will remove 1 "/" from "http://" as it's consider that's
		// file system path. So we better use normal string concat
		issueURL := project.WebURL + "/-/issues"
		if num > 0 {
			issueURL = issueURL + "/" + strconv.FormatInt(num, 10)
		}

		err = browse(issueURL)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	issueCmd.AddCommand(issueBrowseCmd)
	carapace.Gen(issueBrowseCmd).PositionalCompletion(
		action.Remotes(),
		action.Issues(issueList),
	)
}
