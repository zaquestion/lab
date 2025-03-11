package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// ciStatusCmd represents the run command
var ciStatusCmd = &cobra.Command{
	Use:     "status [branch]",
	Aliases: []string{"run"},
	Short:   "Textual representation of a CI pipeline",
	Example: heredoc.Doc(`
		lab ci status
		lab ci status upstream 608 --merge-request
		lab ci status 600 --wait
		lab ci status upstream 125 --merge-request --bridge 'security-tests'`),
	RunE:             nil,
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			rn  string
			err error
		)

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

		rn, pipelineID, err := getPipelineFromArgs(args, forMR)
		if err != nil {
			log.Fatal(err)
		}

		pid := rn

		pager := newPager(cmd.Flags())
		defer pager.Close()

		w := tabwriter.NewWriter(os.Stdout, 2, 4, 1, byte(' '), 0)

		wait, err := cmd.Flags().GetBool("wait")
		if err != nil {
			log.Fatal(err)
		}

		var jobStructList []lab.JobStruct
		jobs := make([]*gitlab.Job, 0)

		fmt.Fprintln(w, "Stage:\tName\t-\tStatus")
		for {
			// fetch all of the CI Jobs from the API
			jobStructList, err = lab.CIJobs(pid, pipelineID, followBridge, bridgeName)
			if err != nil {
				log.Fatal(errors.Wrap(err, "failed to find ci jobs"))
			}

			for _, jobStruct := range jobStructList {
				jobs = append(jobs, jobStruct.Job)
			}

			// filter out old jobs
			jobs = latestJobs(jobs)
			if len(jobs) == 0 {
				log.Fatal("no CI jobs found in pipeline ", pipelineID, " on remote ", rn)
				return
			}

			// print the status of all current jobs
			for _, job := range jobs {
				jobStatus := color.YellowString(job.Status)
				switch job.Status {
				case "failed":
					jobStatus = color.RedString(job.Status)
				case "cancelled":
					jobStatus = color.RedString(job.Status)
				case "success":
					jobStatus = color.GreenString(job.Status)
				}
				fmt.Fprintf(w, "%s:\t%s\t-\t%s\n", job.Stage, job.Name, jobStatus)
			}

			dontWaitForJobsToFinish := !wait ||
				(jobs[0].Pipeline.Status != "pending" &&
					jobs[0].Pipeline.Status != "running")
			if dontWaitForJobsToFinish {
				break
			}

			fmt.Fprintln(w)

			// don't spam the api TOO much
			time.Sleep(10 * time.Second)
		}

		mrPipelineStatus := color.YellowString(jobs[0].Pipeline.Status)
		switch jobs[0].Pipeline.Status {
		case "failed":
			mrPipelineStatus = color.RedString(jobs[0].Pipeline.Status)
		case "cancelled":
			mrPipelineStatus = color.RedString(jobs[0].Pipeline.Status)
		case "success":
			mrPipelineStatus = color.GreenString(jobs[0].Pipeline.Status)
		}

		fmt.Fprintf(w, "\nPipeline Status: %s\n", mrPipelineStatus)
		// exit w/ status code 1 to indicate a job failure
		if wait && jobs[0].Pipeline.Status != "success" {
			os.Exit(1)
		}
		w.Flush()
	},
}

func init() {
	ciStatusCmd.Flags().Bool("wait", false, "continuously print the status and wait to exit until the pipeline finishes. Exit code indicates pipeline status")
	ciStatusCmd.Flags().Bool("merge-request", false, "use merge request pipeline if enabled")
	ciCmd.AddCommand(ciStatusCmd)

	carapace.Gen(ciStatusCmd).PositionalCompletion(
		action.Remotes(),
		action.RemoteBranches(0),
	)
}
