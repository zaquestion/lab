package cmd

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	color "github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
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
		noSkipped, err := cmd.Flags().GetBool("no-skipped")
		if err != nil {
			log.Fatal(err)
		}
		useColor, err := cmd.Flags().GetBool("color")
		if err != nil {
			log.Fatal(err)
		}
		onlyFailures, err := cmd.Flags().GetBool("failures")
		if err != nil {
			log.Fatal(err)
		}

		fmt.Fprintln(w, "Stage:\tName\t-\tStatus")
		jobFormat := "%s:\t%s\t(%d)\t-\t%s\n"
		failed := color.New(color.FgRed)
		passed := color.New(color.FgGreen)
		skipped := color.New(color.FgYellow)
		running := color.New(color.FgBlue)
		created := color.New(color.FgMagenta)
		defaultPrinter := color.New(color.FgBlack)
		defaultPrinter.DisableColor()
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
					switch job.Status {
					case "failed":
						printer = failed
					case "success":
						printer = passed
					case "running":
						printer = running
					case "created":
						printer = created
					case "skipped":
						printer = skipped
					default:
						printer = defaultPrinter
					}
					printer.Fprintf(w, jobFormat, job.Stage, job.Name, job.ID, job.Status)
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

func init() {
	ciStatusCmd.MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote_branches")
	ciStatusCmd.Flags().Bool("wait", false, "Continuously print the status and wait to exit until the pipeline finishes. Exit code indicates pipeline status")
	ciStatusCmd.Flags().Bool("no-skipped", false, "Ignore skipped tests - do not print them")
	ciStatusCmd.Flags().Bool("failures", false, "Only print failures")
	ciStatusCmd.Flags().Bool("color", false, "Use color for success and failure")
	ciCmd.AddCommand(ciStatusCmd)
}
