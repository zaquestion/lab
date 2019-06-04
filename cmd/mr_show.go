package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
	zsh "github.com/rsteube/cobra-zsh-gen"
)

var mrShowCmd = &cobra.Command{
	Use:        "show [remote] <id>",
	Aliases:    []string{"get"},
	ArgAliases: []string{"s"},
	Short:      "Describe a merge request",
	Long:       ``,
	Run: func(cmd *cobra.Command, args []string) {
		rn, mrNum, err := parseArgs(args)
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
Assignee: %s
Author: %s
Milestone: %s
Labels: %s
WebURL: %s
`,
		mr.IID, mr.Title, mr.Description, project, mr.SourceBranch,
		mr.TargetBranch, state, assignee,
		mr.Author.Username, milestone, labels, mr.WebURL)
}

func init() {
	zsh.Wrap(mrShowCmd).MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	zsh.Wrap(mrShowCmd).MarkZshCompPositionalArgumentCustom(2, "__lab_completion_merge_request $words[2]")
	mrCmd.AddCommand(mrShowCmd)
}
