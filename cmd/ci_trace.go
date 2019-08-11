package cmd

import (
	bytes "bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	cacheKey       string
	writeToCache   bool = false
	cachedResponse bytes.Buffer
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
			if err != nil {
				log.Fatal(args[0], " is not a remote:", err)
			} else if !ok {
				i, err := strconv.Atoi(args[0])
				if err == nil {
					rn, err := git.PathWithNameSpace(remote)
					if err != nil {
						log.Fatal(err)
					}
					project, err := lab.FindProject(rn)
					if err != nil {
						log.Fatal(err)
					}
					doTraceByJobID(context.Background(), os.Stdout, project.ID, i)
					if writeToCache {
						lab.WriteCache(cacheKey, cachedResponse.Bytes())
					}
					return
				} else {
					log.Fatal(args[0], " is not a remote:", err)
				}
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
		err = doTrace(context.Background(), os.Stdout, project.ID, branch, jobName)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func doTraceByJobID(ctx context.Context, w io.Writer, pid interface{}, jobID int) error {
	var (
		offset int64
	)
	client := lab.Client()
	offset = 0
	job, _, err := client.Jobs.GetJob(pid, jobID)
	if err != nil {
		return err
	}
	cacheKey = fmt.Sprintf("cmd_trace-%d-%d.log", jobID, job.CreatedAt.Unix())
	var reader io.Reader

	inCache, cached, err := lab.ReadCache(cacheKey)

	if jobIsFinished(job) && inCache && err == nil {
		fmt.Fprintf(w, "[FROM CACHE]")
		reader = bytes.NewReader(cached)
	} else {
		trace, _, err := client.Jobs.GetTraceFile(pid, jobID)
		if err != nil {
			return err
		}
		reader = io.TeeReader(trace, &cachedResponse)
		writeToCache = true
	}

	fmt.Fprintf(w, "Showing logs for %s job #%d\n", job.Name, job.ID)
	return printTrace(w, &offset, reader)
}

func printTrace(w io.Writer, offset *int64, trace io.Reader) error {
	_, err := io.CopyN(ioutil.Discard, trace, *offset)
	lenT, err := io.Copy(w, trace)
	if err != nil {
		return err
	}
	*offset += int64(lenT)
	return nil
}

func doTrace(ctx context.Context, w io.Writer, pid interface{}, branch, name string) error {
	var (
		once   sync.Once
		offset *int64
	)
	*offset = 0
	for range time.NewTicker(time.Second * 3).C {
		trace, job, err := lab.CITrace(pid, branch, name)
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
		if ctx.Err() == context.Canceled {
			break
		}
		printTrace(w, offset, trace)
		if jobIsFinished(job) {
			return nil
		}
	}
	return nil
}

func jobIsFinished(job *gitlab.Job) bool {
	if job.Status == "success" ||
		job.Status == "failed" ||
		job.Status == "skipped" ||
		job.Status == "cancelled" {
		return true
	}
	return false
}

func init() {
	ciTraceCmd.MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	ciTraceCmd.MarkZshCompPositionalArgumentCustom(2, "__lab_completion_remote_branches $words[2]")
	ciCmd.AddCommand(ciTraceCmd)
}
