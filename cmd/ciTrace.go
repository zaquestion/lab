package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// ciLintCmd represents the lint command
var ciTraceCmd = &cobra.Command{
	Use:     "trace [remote [[branch:]job]]",
	Aliases: []string{"logs"},
	Short:   "Trace the output of a ci job",
	Long:    `If a job is not specified the latest running job or last job in the pipeline is used`,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			remote  string
			jobName string
		)

		branch, err := git.CurrentBranch()
		if err != nil {
			log.Fatal(err)
		}
		if len(args) > 1 {
			jobName = args[1]
			if strings.Contains(args[1], ":") {
				ps := strings.Split(args[1], ":")
				branch, jobName = ps[0], ps[1]
			}
		}
		remote = determineSourceRemote(branch)
		if len(args) > 0 {
			ok, err := git.IsRemote(args[0])
			if err != nil || !ok {
				log.Fatal(args[0], "is not a remote:", err)
			}
			remote = args[0]
		}

		rn, err := git.PathWithNameSpace(remote)
		if err != nil {
			log.Fatal(err)
		}
		project, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}
		var (
			once   sync.Once
			offset int64
		)
	FOR:
		for range time.NewTicker(time.Second * 3).C {
			trace, job, err := lab.CITrace(project.ID, branch, jobName)
			if job == nil {
				log.Fatal(errors.Wrap(err, "failed to find job"))
			}
			switch job.Status {
			case "pending":
				fmt.Printf("%s is pending...\n", job.Name)
				continue
			case "manual":
				fmt.Printf("Manual job %s not started\n", job.Name)
				break FOR
			}
			once.Do(func() {
				if jobName == "" {
					jobName = job.Name
				}
				fmt.Printf("Showing logs for %s job #%d\n", job.Name, job.ID)
			})
			buf, err := ioutil.ReadAll(trace)
			if err != nil {
				log.Fatal(err)
			}
			r := bytes.NewReader(buf)
			r.Seek(offset, io.SeekStart)
			new, err := ioutil.ReadAll(r)

			offset += int64(len(new))
			fmt.Print(string(new))
			if job.Status == "success" ||
				job.Status == "failed" ||
				job.Status == "cancelled" {
				break
			}
		}
	},
}

func init() {
	ciCmd.AddCommand(ciTraceCmd)
}
