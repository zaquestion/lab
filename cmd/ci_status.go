package cmd

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	color "github.com/fatih/color"
        flag "github.com/spf13/pflag"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
  onlyFailures bool
  useColor bool
  noSkipped bool
  wait bool
  jobFormat = "%s:\t%s\t-\t%s\tid: %d\n"
  failed = color.New(color.FgRed)
  passed = color.New(color.FgGreen)
  skipped = color.New(color.FgYellow)
  running = color.New(color.FgBlue)
  created = color.New(color.FgMagenta)
  defaultPrinter = color.New(color.FgBlack)
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

		fmt.Fprintln(w, "Stage:\tName\t-\tStatus")
		color.NoColor = !useColor
		var (
			printer *color.Color
		)
		for {
			for _, job := range jobs {
				if noSkipped && job.Status == "skipped" {
					continue
				} else if onlyFailures && job.Status != "failed" {
					continue
				} else {
                                        printer = statusColor(job.Status)
					printer.Fprintf(w, jobFormat, job.Stage, job.Name, job.Status, job.ID)
				}
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
        case "running":
                return running
        case "created":
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
        ciStatusCmd.Flags().SetNormalizeFunc(aliasFailures)
	ciCmd.AddCommand(ciStatusCmd)
}
