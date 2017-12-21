package cmd

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrShowCmd = &cobra.Command{
	Use:        "show [remote]",
	Aliases:    []string{"get"},
	ArgAliases: []string{"s"},
	Short:      "Describe a merge request",
	Long:       ``,
	Run: func(cmd *cobra.Command, args []string) {
		remote, mrNum, err := parseArgsRemote(args)
		if err != nil {
			log.Fatal(err)
		}
		if remote == "" {
			remote = forkedFromRemote
		}
		rn, err := git.PathWithNameSpace(remote)
		if err != nil {
			log.Fatal(err)
		}

		mr, err := lab.MRGet(rn, int(mrNum))
		if err != nil {
			log.Fatal(err)
		}

		printMR(mr, rn)
	},
}

func printMR(mr *gitlab.MergeRequest, project string) {
	assignee := "None"
	milestone := "None"
	labels := "None"
	state := map[string]string{
		"opened": "Open",
		"closed": "Closed",
		"merged": "Merged",
	}[mr.State]
	if mr.Assignee.Username != "" {
		assignee = mr.Assignee.Username
	}
	if mr.Milestone != nil {
		milestone = mr.Milestone.Title
	}
	if len(mr.Labels) > 0 {
		labels = strings.Join(mr.Labels, ", ")
	}

	fmt.Printf(`
#%d %s
===================================
%s
-----------------------------------
Project: %s
Branches: %s->%s
Status: %s
Work in Progress: %s
Assignee: %s
Author: %s
Milestone: %s
Labels: %s
WebURL: %s
`,
		mr.IID, mr.Title, mr.Description, project, mr.SourceBranch,
		mr.TargetBranch, state, strconv.FormatBool(mr.WorkInProgress), assignee,
		mr.Author.Username, milestone, labels, mr.WebURL)
}

func init() {
	mrCmd.AddCommand(mrShowCmd)
}
