package cmd

import (
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var issueCloseCmd = &cobra.Command{
	Use:              "close [remote] <id>",
	Aliases:          []string{"delete"},
	Short:            "Close issue by ID",
	Args:             cobra.MinimumNArgs(1),
	PersistentPreRun: labPersistentPreRun,
	Example: heredoc.Doc(`
		lab issue close 1234
		lab issue close origin --duplicate 123 1234
		lab issue close --duplicate other-project#123 1234`),
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsRemoteAndID(args)
		if err != nil {
			log.Fatal(err)
		}

		p, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}

		dupID, _ := cmd.Flags().GetString("duplicate")
		if dupID != "" {
			if !strings.Contains(dupID, "#") {
				dupID = "#" + dupID
			}
			err = lab.IssueDuplicate(p.ID, int(id), dupID)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Issue #%d closed as duplicate of %s\n", id, dupID)
		} else {
			err = lab.IssueClose(p.ID, int(id))
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Issue #%d closed\n", id)
		}
	},
}

func init() {
	issueCloseCmd.Flags().StringP("duplicate", "", "", "mark as duplicate of another issue")
	issueCmd.AddCommand(issueCloseCmd)
	carapace.Gen(issueCloseCmd).PositionalCompletion(
		action.Remotes(),
		action.Issues(issueList),
	)
}
