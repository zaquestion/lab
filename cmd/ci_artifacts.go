package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var ciArtifactsCmd = &cobra.Command{
	Use:   "artifacts [remote] [branch][:job]",
	Short: "Download artifacts of a ci job",
	Long: heredoc.Doc(`
		Download the CI pipeline job artifacts for the given or current branch if
		none provided.

		The branch name, when using with the --merge-request option, can be the
		merge request number, which matches the branch name internally.	The "job"
		portion is the given job name, which may contain whitespace characters
		and which, for this specific case, must be quoted.`),
	Example: heredoc.Doc(`
		lab ci artifacts upstream feature_branch
		lab ci artifacts upstream :'my custom stage'
		lab ci artifacts upstream 125 --merge-request
		lab ci artifacts upstream 125:'my custom stage' --merge-request
		lab ci artifacts upstream 125:'build' --merge-request --bridge 'security-tests'`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			rn      string
			jobName string
			err     error
		)
		jobName, branchArgs, err := filterJobArg(args)
		if err != nil {
			log.Fatal(err)
		}

		forMR, err := cmd.Flags().GetBool("merge-request")
		if err != nil {
			log.Fatal(err)
		}

		bridgeName, err = cmd.Flags().GetString("bridge")
		if err != nil {
			log.Fatal(err)
		} else if bridgeName != "" {
			followBridge = true
		} else {
			followBridge, err = cmd.Flags().GetBool("follow")
			if err != nil {
				log.Fatal(err)
			}
		}

		path, err := cmd.Flags().GetString("artifact-path")
		if err != nil {
			log.Fatal(err)
		}

		onlyPassed, err := cmd.Flags().GetBool("passed")
		if err != nil {
			log.Fatal(err)
		}

		rn, pipelineID, err = getPipelineFromArgs(branchArgs, forMR, onlyPassed)
		if err != nil {
			log.Fatal(err)
		}

		project, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}
		projectID := project.ID

		r, outpath, err := lab.CIArtifacts(projectID, pipelineID, jobName, path, followBridge, bridgeName)
		if err != nil {
			log.Fatal(err)
		}

		dst, err := os.Create(outpath)
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.Copy(dst, r)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Downloaded %s\n", outpath)
	},
}

func init() {
	ciArtifactsCmd.Flags().Bool("merge-request", false, "use merge request pipeline if enabled")
	ciArtifactsCmd.Flags().StringP("artifact-path", "p", "", "only download specified file from archive")
	ciArtifactsCmd.Flags().Bool("passed", false, "consider only pipeline that succeeded")
	ciCmd.AddCommand(ciArtifactsCmd)
	carapace.Gen(ciArtifactsCmd).PositionalCompletion(
		action.Remotes(),
		action.RemoteBranches(0),
	)
}
