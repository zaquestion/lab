package cmd

import (
	"fmt"
	"strconv"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var issueMoveCmd = &cobra.Command{
	Use:              "move <id> <destination>",
	Short:            "Move issue to another project",
	Long:             ``,
	Example:          "lab issue move 5 zaquestion/test/           # FQDN must match",
	Args:             cobra.MinimumNArgs(2),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		// get this rn
		srcRN, err := getRemoteName(defaultRemote)
		if err != nil {
			log.Fatal(err)
		}

		// get the issue ID
		id, err := strconv.Atoi(args[0])
		if err != nil {
			log.Fatal(err)
		}

		// get the dest rn (everything after gitlab.com/, for example,
		destRN := args[1]

		issueURL, err := lab.MoveIssue(srcRN, id, destRN)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Issue moved to:", issueURL)
	},
}

func init() {
	issueCmd.AddCommand(issueMoveCmd)
	carapace.Gen(issueMoveCmd).PositionalCompletion(
		action.Remotes(),
		action.Issues(issueList),
	)
}
