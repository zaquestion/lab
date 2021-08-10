package cmd

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/pkg/errors"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// ciLintCmd represents the lint command
var ciTraceCmd = &cobra.Command{
	Use:     "trace [remote] [branch[:job]]",
	Aliases: []string{"logs"},
	Short:   "Trace the output of a ci job",
	Long: heredoc.Doc(`
		Download the CI pipeline job artifacts for the given or current branch if
		none provided. If a job is not specified the latest running job or last
		job in the pipeline is used

		The branch name, when using with the --merge-request option, can be the
		merge request number, which matches the branch name internally.	The "job"
		portion is the given job name, which may contain whitespace characters
		and which, for this specific case, must be quoted.`),
	Example: heredoc.Doc(`
		lab ci trace upstream feature_branch
		lab ci trace upstream 18 --merge-request
		lab ci trace upstream 18:'my custom stage' --merge-request
		lab ci trace upstream 18:'my custom stage' --merge-request --bridge 'security-tests'`),
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

		rn, pipelineID, err := getPipelineFromArgs(branchArgs, forMR)
		if err != nil {
			log.Fatal(err)
		}

		project, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}
		projectID = project.ID

		pager := newPager(cmd.Flags())
		defer pager.Close()

		err = doTrace(context.Background(), os.Stdout, projectID, pipelineID, jobName)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func doTrace(ctx context.Context, w io.Writer, pid interface{}, pipelineID int, name string) error {
	var (
		once   sync.Once
		offset int64
	)
	for range time.NewTicker(time.Second * 3).C {
		if ctx.Err() == context.Canceled {
			break
		}
		trace, job, err := lab.CITrace(pid, pipelineID, name, followBridge, bridgeName)
		if err != nil || job == nil || trace == nil {
			return errors.Wrap(err, "failed to find job")
		}
		switch job.Status {
		case "pending":
			fmt.Fprintf(w, "%s is pending... waiting for job to start\n", job.Name)
			continue
		case "manual":
			fmt.Fprintf(w, "Manual job %s not started, waiting for job to start\n", job.Name)
			continue
		}
		once.Do(func() {
			if name == "" {
				name = job.Name
			}
			fmt.Fprintf(w, "Showing logs for %s job #%d\n", job.Name, job.ID)
		})

		_, err = io.CopyN(ioutil.Discard, trace, offset)
		if err != nil {
			return err
		}

		lenT, err := io.Copy(w, trace)
		if err != nil {
			return err
		}
		offset += int64(lenT)

		if job.Status == "success" ||
			job.Status == "failed" ||
			job.Status == "cancelled" {
			return nil
		}
	}
	return nil
}

func filterJobArg(args []string) (string, []string, error) {
	branchArgs := []string{}
	jobName := ""

	if len(args) == 1 {
		ok, err := git.IsRemote(args[0])
		if err != nil {
			return "", branchArgs, err
		}
		if ok {
			branchArgs = append(branchArgs, args[0])
		} else {
			jobName = args[0]
		}
	} else if len(args) > 1 {
		branchArgs = append(branchArgs, args[0])
		jobName = args[1]
	}

	if strings.Contains(jobName, ":") {
		ps := strings.SplitN(jobName, ":", 2)
		branchArgs = append(branchArgs, ps[0])
		jobName = ps[1]
	}

	return jobName, branchArgs, nil
}

func init() {
	ciTraceCmd.Flags().Bool("merge-request", false, "use merge request pipeline if enabled")
	ciCmd.AddCommand(ciTraceCmd)
	carapace.Gen(ciTraceCmd).PositionalCompletion(
		action.Remotes(),
		action.RemoteBranches(0),
	)
}
