package cmd

import (
	"fmt"
	"log"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// listCmd represents the list command
var checkoutCmd = &cobra.Command{
	Use:   "checkout",
	Short: "Checkout an open merge request",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rn, err := git.PathWithNameSpace(forkedFromRemote)
		if err != nil {
			log.Fatal(err)
		}
		mrIDStr := args[0]
		mrID, err := strconv.Atoi(mrIDStr)
		if err != nil {
			log.Fatal(err)
		}
		mrs, err := lab.ListMRs(rn, &gitlab.ListProjectMergeRequestsOptions{
			IIDs: []int{mrID},
		})
		if err != nil {
			log.Fatal(err)
		}
		if len(mrs) < 1 {
			fmt.Printf("MR #%s not found\n", mrIDStr)
			return
		}
		// https://docs.gitlab.com/ee/user/project/merge_requests/#checkout-merge-requests-locally
		branch := mrs[0].SourceBranch
		mr := fmt.Sprintf("refs/merge-requests/%s/head", mrIDStr)
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
	mrCmd.AddCommand(checkoutCmd)
}
