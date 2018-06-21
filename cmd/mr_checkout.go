package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/tcnksm/go-gitconfig"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// mrCheckoutConfig holds configuration values for calls to lab mr checkout
type mrCheckoutConfig struct {
	branch string
	track  bool
}

var (
	mrCheckoutCfg mrCheckoutConfig
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

		mr := mrs[0]
		// If the config does not specify a branch, use the mr source branch name
		if mrCheckoutCfg.branch == "" {
			mrCheckoutCfg.branch = mr.SourceBranch
		}
		// By default, fetch to configured branch
		fetchToRef := mrCheckoutCfg.branch

		// If track, make sure we have a remote for the mr author and then set
		// the fetchToRef to the mr author/sourceBranch
		if mrCheckoutCfg.track {
			// Check if remote already exists
			if _, err := gitconfig.Local("remote." + mr.Author.Username + ".url"); err != nil {
				// Find and create remote
				mrProject, err := lab.GetProject(mr.SourceProjectID)
				if err != nil {
					log.Fatal(err)
				}
				if err := git.RemoteAdd(mr.Author.Username, mrProject.SSHURLToRepo, "."); err != nil {
					log.Fatal(err)
				}
			}
			fetchToRef = fmt.Sprintf("refs/remotes/%s/%s", mr.Author.Username, mr.SourceBranch)
		}

		// https://docs.gitlab.com/ce/user/project/merge_requests/#checkout-merge-requests-locally
		mrRef := fmt.Sprintf("refs/merge-requests/%d/head", mrID)
		fetchRefSpec := fmt.Sprintf("%s:%s", mrRef, fetchToRef)
		if err := git.New("fetch", forkedFromRemote, fetchRefSpec).Run(); err != nil {
			log.Fatal(err)
		}

		if mrCheckoutCfg.track {
			// Create configured branch with tracking from fetchToRef
			// git branch --flags <branchname> [<start-point>]
			if err := git.New("branch", "--track", mrCheckoutCfg.branch, fetchToRef).Run(); err != nil {
				log.Fatal(err)
			}
		}

		// Check out branch
		if err := git.New("checkout", mrCheckoutCfg.branch).Run(); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	checkoutCmd.Flags().StringVarP(&mrCheckoutCfg.branch, "branch", "b", "", "checkout merge request with <branch> name")
	checkoutCmd.Flags().BoolVarP(&mrCheckoutCfg.track, "track", "t", false, "set checked out branch to track mr author remote branch, adds remote if needed")
	mrCmd.AddCommand(checkoutCmd)
}
