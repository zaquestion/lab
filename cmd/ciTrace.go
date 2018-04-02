package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// ciLintCmd represents the lint command
var ciTraceCmd = &cobra.Command{
	Use:     "trace [remote [job]]",
	Aliases: []string{"logs"},
	Short:   "Trace the output of a ci job",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			remote  string
			jobName string
		)
		if len(args) > 0 {
			ok, err := git.IsRemote(args[0])
			if err != nil || !ok {
				log.Fatal(args[0], "is not a remote:", err)
			}
			remote = args[0]
		}
		if len(args) > 1 {
			jobName = args[1]
		}
		if remote == "" {
			remote = forkedFromRemote
		}

		// See if we're in a git repo or if global is set to determine
		// if this should be a personal snippet
		rn, err := git.PathWithNameSpace(remote)
		if err != nil {
			log.Fatal(err)
		}
		project, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}
		sha, err := git.Sha("HEAD")
		if err != nil {
			log.Fatal(err)
		}
		var (
			offset int64
			tick   = time.Second * 3
		)
	FOR:
		for range time.NewTicker(tick).C {
			trace, status, err := lab.CITrace(project.ID, sha, jobName)
			switch status {
			case "pending":
				fmt.Println(err)
				continue
			case "manual":
				fmt.Println(err)
				break FOR
			}
			if err != nil {
				log.Fatal(err)
			}
			buf, err := ioutil.ReadAll(trace)
			if err != nil {
				log.Fatal(err)
			}
			r := bytes.NewReader(buf)
			r.Seek(offset, io.SeekStart)
			new, err := ioutil.ReadAll(r)

			offset += int64(len(new))
			fmt.Print(string(new))
			if status == "success" || status == "failed" {
				break
			}
		}
	},
}

func init() {
	ciCmd.AddCommand(ciTraceCmd)
}
