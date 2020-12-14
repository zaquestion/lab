package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitconfig "github.com/tcnksm/go-gitconfig"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
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
	Use:              "checkout [remote] <id>",
	Short:            "Checkout an open merge request",
	Long:             ``,
	Args:             cobra.RangeArgs(1, 2),
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, mrID, err := parseArgsRemoteAndID(args)
		if err != nil {
			log.Fatal(err)
		}
		var targetRemote = forkedFromRemote
		if len(args) == 2 {
			// parseArgs above already validated this is a remote
			targetRemote = args[0]
		}

		mrs, err := lab.MRList(rn, gitlab.ListProjectMergeRequestsOptions{
			IIDs: []int{int(mrID)},
		}, 1)
		if err != nil {
			log.Fatal(err)
		}
		if len(mrs) < 1 {
			fmt.Printf("MR !%d not found\n", mrID)
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
				urlToRepo := labURLToRepo(mrProject)
				if err := git.RemoteAdd(mr.Author.Username, urlToRepo, "."); err != nil {
					log.Fatal(err)
				}
			}
			fetchToRef = fmt.Sprintf("refs/remotes/%s/%s", mr.Author.Username, mr.SourceBranch)
		}

		if err := git.New("show-ref", "--verify", "--quiet", "refs/heads/"+fetchToRef).Run(); err == nil {
			fmt.Println("ERROR: mr", mrID, "branch", fetchToRef, "already exists.")
			os.Exit(1)
		}

		// https://docs.gitlab.com/ce/user/project/merge_requests/#checkout-merge-requests-locally
		mrRef := fmt.Sprintf("refs/merge-requests/%d/head", mrID)
		fetchRefSpec := fmt.Sprintf("%s:%s", mrRef, fetchToRef)
		if err := git.New("fetch", targetRemote, fetchRefSpec).Run(); err != nil {
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
	// useHTTP is defined in "project_create.go"
	checkoutCmd.Flags().BoolVar(&useHTTP, "http", false, "checkout using HTTP protocol instead of SSH")
	mrCmd.AddCommand(checkoutCmd)
	carapace.Gen(checkoutCmd).PositionalCompletion(
		carapace.ActionCallback(func(args []string) carapace.Action {
			return action.MergeRequests(mrList).Invoke([]string{"origin"}).ToA()
		}),
	)
}
