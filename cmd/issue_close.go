package cmd

import (
	"fmt"
	"strings"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var issueCloseCmd = &cobra.Command{
	Use:              "close [remote] <id>",
	Aliases:          []string{"delete"},
	Short:            "Close issue by id",
	Long:             ``,
	Args:             cobra.MinimumNArgs(1),
	PersistentPreRun: LabPersistentPreRun,
	Example: `lab issue close 1234
lab issue close --duplicate 123 1234
lab issue close --duplicate other-project#123 1234`,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsRemoteAndID(args)
		if err != nil {
			log.Fatal(err)
		}

		p, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}

		dupId, _ := cmd.Flags().GetString("duplicate")
		if dupId != "" {
			if !strings.Contains(dupId, "#") {
				dupId = "#" + dupId
			}
			err = lab.IssueDuplicate(p.ID, int(id), dupId)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Issue #%d closed as duplicate of %s\n", id, dupId)
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
