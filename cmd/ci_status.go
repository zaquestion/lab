package cmd

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
	zsh "github.com/rsteube/cobra-zsh-gen"
)

// ciStatusCmd represents the run command
var ciStatusCmd = &cobra.Command{
	Use:     "status [branch]",
	Aliases: []string{"run"},
	Short:   "Textual representation of a CI pipeline",
	Long:    ``,
	Example: `lab ci status
lab ci status --wait`,
	RunE: nil,
	Run: func(cmd *cobra.Command, args []string) {
		branch, err := git.CurrentBranch()
		if err != nil {
			log.Fatal(err)
		}

		if len(args) > 1 {
			branch = args[1]
		}
		remote := determineSourceRemote(branch)
		if len(args) > 0 {
			ok, err := git.IsRemote(args[0])
			if err != nil || !ok {
				log.Fatal(args[0], " is not a remote:", err)
			}
			remote = args[0]
		}
		rn, err := git.PathWithNameSpace(remote)
		if err != nil {
			log.Fatal(err)
		}
		pid := rn

		w := tabwriter.NewWriter(os.Stdout, 2, 4, 1, byte(' '), 0)
		jobs, err := lab.CIJobs(pid, branch)
		if err != nil {
			log.Fatal(errors.Wrap(err, "failed to find ci jobs"))
		}
		jobs = latestJobs(jobs)

		if len(jobs) == 0 {
			return
		}

		wait, err := cmd.Flags().GetBool("wait")
		if err != nil {
			log.Fatal(err)
		}

		fmt.Fprintln(w, "Stage:\tName\t-\tStatus")
		for {
			for _, job := range jobs {
				fmt.Fprintf(w, "%s:\t%s\t-\t%s\n", job.Stage, job.Name, job.Status)
			}
			if !wait {
				break
			}
			if jobs[0].Pipeline.Status != "pending" &&
				jobs[0].Pipeline.Status != "running" {
				break
			}
			fmt.Fprintln(w)
		}

		fmt.Fprintf(w, "\nPipeline Status: %s\n", jobs[0].Pipeline.Status)
		if wait && jobs[0].Pipeline.Status != "success" {
			os.Exit(1)
		}
		w.Flush()
	},
}

func init() {
	zsh.Wrap(ciStatusCmd).MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote_branches")
	ciStatusCmd.Flags().Bool("wait", false, "Continuously print the status and wait to exit until the pipeline finishes. Exit code indicates pipeline status")
	ciCmd.AddCommand(ciStatusCmd)
}
