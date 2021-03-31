package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var ciArtifactsCmd = &cobra.Command{
	Use:              "artifacts [remote [[branch:]job]]",
	Short:            "Download artifacts of a ci job",
	Long:             `If a job is not specified the latest job with artifacts is used`,
	PersistentPreRun: LabPersistentPreRun,
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

		followBridge, err = cmd.Flags().GetBool("follow")
		if err != nil {
			log.Fatal(err)
		}

		path, err := cmd.Flags().GetString("artifact-path")
		if err != nil {
			log.Fatal(err)
		}

		rn, pipelineID, err := getPipelineFromArgs(branchArgs, forMR)
		if err != nil {
			log.Fatal(err)
		}

		project, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}
		projectID := project.ID

		r, outpath, err := lab.CIArtifacts(projectID, pipelineID, jobName, path, followBridge)
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
	ciCmd.AddCommand(ciArtifactsCmd)
	carapace.Gen(ciArtifactsCmd).PositionalCompletion(
		action.Remotes(),
		action.RemoteBranches(0),
	)
}
