package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	branch string
)

// listCmd represents the list command
var checkoutCmd = &cobra.Command{
	Use:   "checkout",
	Short: "Checkout an open merge request",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rn, mrID, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		mrs, err := lab.MRList(rn, &gitlab.ListProjectMergeRequestsOptions{
			IIDs: []int{int(mrID)},
		})
		if err != nil {
			log.Fatal(err)
		}
		if len(mrs) < 1 {
			fmt.Printf("MR #%d not found\n", mrID)
			return
		}
		// https://docs.gitlab.com/ee/user/project/merge_requests/#checkout-merge-requests-locally
		if branch == "" {
			branch = mrs[0].SourceBranch
		}
		mr := fmt.Sprintf("refs/merge-requests/%d/head", mrID)
		gitf := git.New("fetch", forkedFromRemote, fmt.Sprintf("%s:%s", mr, branch))
		err = gitf.Run()
		if err != nil {
			log.Fatal(err)
		}

		gitc := git.New("checkout", branch)
		err = gitc.Run()
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	checkoutCmd.Flags().StringVarP(&branch, "branch", "b", "", "checkout merge request with <branch> name")
	mrCmd.AddCommand(checkoutCmd)
}
