package cmd

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	color "github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	onlyFailures   bool
	useColor       bool
	noSkipped      bool
	wait           bool
	noCreated      bool
	summaryOnly    bool
	failed         = color.New(color.FgRed)
	passed         = color.New(color.FgGreen)
	skipped        = color.New(color.FgYellow)
	running        = color.New(color.FgBlue)
	created        = color.New(color.FgMagenta)
	defaultPrinter = color.New(color.FgBlack)
)

const (
	jobFormat = "%*s: %*s - %10s id: %d\n"
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
	Run:  runCommand,
}

func runCommand(cmd *cobra.Command, args []string) {
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

	if !summaryOnly {
		fmt.Fprintln(w, "Stage:\tName\t-\tStatus")
	}
	color.NoColor = !useColor
	var (
		printer *color.Color
	)
	pipelineId := jobs[0].Pipeline.ID
	pipeline, _, err := lab.Client().Pipelines.GetPipeline(pid, pipelineId)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to get pipeline information"))
	}
	maxNameLength := 0
	maxStageLength := 0
	for _, job := range jobs {
		if len(job.Name) > maxNameLength {
			maxNameLength = len(job.Name)
		}
		if len(job.Stage) > maxStageLength {
			maxStageLength = len(job.Stage)
		}
	}

	for {
		if !summaryOnly {
			for _, job := range jobs {
				if noSkipped && job.Status == "skipped" {
					continue
				} else if onlyFailures && job.Status != "failed" {
					continue
				} else if noCreated && job.Status == "created" {
					continue
				} else {
					printer = statusColor(job.Status)
					printer.Fprintf(w, jobFormat, maxStageLength, job.Stage, -maxNameLength, job.Name, job.Status, job.ID)
				}
			}
		}
		if !wait {
			break
		}
		pl, _, err := lab.Client().Pipelines.GetPipeline(pid, pipelineId)
		if err != nil {
			log.Fatal(errors.Wrap(err, "failed to get pipeline information"))
		}
		pipeline = pl
		if pipeline.Status != "pending" && pipeline.Status != "running" {
			break
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, pipelineStatus(pipeline, jobs))
	if wait && pipeline.Status != "success" {
		os.Exit(1)
	}
	w.Flush()
}

func pipelineStatus(pipeline *gitlab.Pipeline, jobs []*gitlab.Job) string {
	return fmt.Sprintf("\nPipeline Status:\t%s\n%s\n\n%s\n",
		statusColor(pipeline.Status).Sprintf(pipeline.Status), timeMessage(pipeline), jobSummary(jobs))
}

func jobSummary(jobs []*gitlab.Job) string {
	numPassed := 0
	totalJobs := 0
	numQueued := 0
	numFailed := 0

	for _, job := range jobs {
		totalJobs++
		if job.Status == "success" {
			numPassed++
		}
		if job.Status == "created" {
			numQueued++
		}
		if job.Status == "failed" {
			numFailed++
		}
	}

	return fmt.Sprintf("total\tpassed\tfailed\tqueued\n%d\t%d\t%d\t%d",
		totalJobs, numPassed, numFailed, numQueued)
}

func timeMessage(pipeline *gitlab.Pipeline) string {
	if pipeline.Status == "pending" {
		return fmt.Sprintf("created at %v", pipeline.CreatedAt)
	} else if pipeline.Status == "running" {
		return fmt.Sprintf("started at %v", pipeline.StartedAt)
	} else {
		hours := pipeline.Duration / (60 * 60)
		minutes := (pipeline.Duration / 60) - (hours * 60)
		seconds := pipeline.Duration % 60
		if hours > 0 {
			return fmt.Sprintf("duration:\t%d hrs, %d min, %d secs", hours, minutes, seconds)
		} else if minutes > 0 {
			return fmt.Sprintf("duration:\t%d min, %d secs", minutes, seconds)
		} else {
			return fmt.Sprintf("duration:\t%d secs", pipeline.Duration)
		}
	}
}

func aliasFailures(f *flag.FlagSet, name string) flag.NormalizedName {
	switch name {
	case "failed":
		name = "failures"
		break
	}
	return flag.NormalizedName(name)
}

func statusColor(status string) *color.Color {
	switch status {
	case "failed":
		return failed
	case "success":
		return passed
	case "passed":
		return passed
	case "running":
		return running
	case "created":
		return created
	case "pending":
		return created
	case "skipped":
		return skipped
	default:
		return defaultPrinter
	}
}

func init() {
	defaultPrinter.DisableColor()
	ciStatusCmd.MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote_branches")
	ciStatusCmd.Flags().BoolVarP(&wait, "wait", "w", false, "Continuously print the status and wait to exit until the pipeline finishes. Exit code indicates pipeline status")
	ciStatusCmd.Flags().BoolVarP(&noSkipped, "no-skipped", "", false, "Ignore skipped tests - do not print them")
	ciStatusCmd.Flags().BoolVarP(&useColor, "color", "c", false, "Use color for success and failure")
	ciStatusCmd.Flags().BoolVarP(&onlyFailures, "failures", "f", false, "Only print failures")
	ciStatusCmd.Flags().BoolVarP(&noCreated, "results-only", "r", false, "Only show completed and running tests. Does not report queued jobs")
	ciStatusCmd.Flags().BoolVarP(&summaryOnly, "summary", "s", false, "Do not show individual jobs, just the pipeline summary")
	ciStatusCmd.Flags().SetNormalizeFunc(aliasFailures)
	ciCmd.AddCommand(ciStatusCmd)
}
