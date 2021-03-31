package cmd

import (
	"github.com/spf13/cobra"
	lab "github.com/zaquestion/lab/internal/gitlab"

	"fmt"
)

var mrDeleteCmd = &cobra.Command{
	Use:              "delete [remote] <id>",
	Aliases:          []string{"del"},
	Short:            "Delete a merge request on GitLab",
	Long:             `Delete a merge request (default: MR created on default branch of origin)`,
	Args:             cobra.MaximumNArgs(2),
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		remote, id, err := parseArgsWithGitBranchMR(args)
		if err != nil {
			log.Fatal(err)
		}
		mrNum := int(id)

		err = lab.MRDelete(remote, mrNum)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Merge request #%d deleted\n", mrNum)
	},
}

func init() {
	mrCmd.AddCommand(mrDeleteCmd)
}
