package cmd

import (
	"log"

	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// ciRunCmd represents the run command
var ciRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the CI pipeline",
	Long:  `Run the CI pipeline for the current branch`,
	Run: func(cmd *cobra.Command, args []string) {
		branch, err := git.CurrentBranch()
		if err != nil {
			log.Fatal(err)
		}
		remote := determineSourceRemote(branch)

		or, err := git.PathWithNameSpace(remote)
		if err != nil {
			log.Fatal(err)
		}

		p, err := lab.CICreate(or, &gitlab.CreatePipelineOptions{Ref: &branch})
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Pipeline started: %d", p.ID)
	},
}

func init() {
	ciCmd.AddCommand(ciRunCmd)
}
