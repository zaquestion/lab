package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	mrShowPatch        bool
	mrShowPatchReverse bool
	mrShowNoColorDiff  bool
)

var mrShowCmd = &cobra.Command{
	Use:              "show [remote] <id>",
	Aliases:          []string{"get"},
	ArgAliases:       []string{"s"},
	Short:            "Describe a merge request",
	Long:             ``,
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {

		rn, mrNum, err := parseArgsWithGitBranchMR(args)
		if err != nil {
			log.Fatal(err)
		}

		mr, err := lab.MRGet(rn, int(mrNum))
		if err != nil {
			log.Fatal(err)
		}

		noMarkdown, _ := cmd.Flags().GetBool("no-markdown")
		if err != nil {
			log.Fatal(err)
		}
		renderMarkdown := !noMarkdown

		if mrShowPatch {
			var remote string

			if len(args) == 1 {
				remote = findLocalRemote(mr.TargetProjectID)
			} else if len(args) == 2 {
				remote = args[0]
			} else {
				log.Fatal("Too many arguments.")
			}

			err := git.Fetch(remote, mr.SHA)
			if err != nil {
				log.Fatal(err)
			}
			git.Show(remote+"/"+mr.TargetBranch, mr.SHA, mrShowPatchReverse)
		} else {
			printMR(mr, rn, renderMarkdown)
		}

		showComments, _ := cmd.Flags().GetBool("comments")
		if showComments {
			discussions, err := lab.MRListDiscussions(rn, int(mrNum))
			if err != nil {
				log.Fatal(err)
			}

			since, err := cmd.Flags().GetString("since")
			if err != nil {
				log.Fatal(err)
			}

			PrintDiscussions(discussions, since, "mr", int(mrNum))
		}
	},
}

func findLocalRemote(ProjectID int) string {
	var remote string

	project, err := lab.GetProject(ProjectID)
	if err != nil {
		log.Fatal(err)
	}
	remotesStr, err := git.GetLocalRemotes()
	if err != nil {
		log.Fatal(err)
	}
	remotes := strings.Split(remotesStr, "\n")

	// find the matching local remote for this project
	for r := range remotes {
		// The fetch and push entries can be different for a remote.
		// Only the fetch entry is useful.
		if strings.Contains(remotes[r], project.SSHURLToRepo+" (fetch)") ||
			strings.Contains(remotes[r], project.HTTPURLToRepo+" (fetch)") {
			found := strings.Split(remotes[r], "\t")
			remote = found[0]
			break
		}
	}

	if remote == "" {
		log.Fatal("remote for ", project.NameWithNamespace, " not found in local remotes")
	}
	return remote
}

func printMR(mr *gitlab.MergeRequest, project string, renderMarkdown bool) {
	assignee := "None"
	milestone := "None"
	labels := "None"
	approvedByUsers := "None"
	state := map[string]string{
		"opened": "Open",
		"closed": "Closed",
		"merged": "Merged",
	}[mr.State]

	if mr.Assignee != nil && mr.Assignee.Username != "" {
		assignee = mr.Assignee.Username
	}
	if mr.Milestone != nil {
		milestone = mr.Milestone.Title
	}
	if len(mr.Labels) > 0 {
		labels = strings.Join(mr.Labels, ", ")
	}

	if renderMarkdown {
		r, _ := glamour.NewTermRenderer(
			glamour.WithStandardStyle("auto"),
		)

		mr.Description, _ = r.Render(mr.Description)
	}

	closingIssues, err := lab.ListIssuesClosedOnMerge(project, mr.IID)
	if err != nil {
		log.Fatal(err)
	}

	approvedBys, err := lab.GetMRApprovedBys(project, mr.IID)
	if err != nil {
		log.Fatal(err)
	}
	if len(approvedBys) > 0 {
		approvedByUsers = strings.Join(approvedBys, ", ")
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
Approved By: %s
Milestone: %s
Labels: %s
Issues Closed by this MR: %s
WebURL: %s
`,
		mr.IID, mr.Title, mr.Description, project, mr.SourceBranch,
		mr.TargetBranch, state, assignee,
		mr.Author.Username, approvedByUsers, milestone, labels,
		strings.Trim(strings.Replace(fmt.Sprint(closingIssues), " ", ",", -1), "[]"),
		mr.WebURL,
	)
}

func init() {
	mrShowCmd.Flags().BoolP("no-markdown", "M", false, "don't use markdown renderer to print the issue description")
	mrShowCmd.Flags().BoolP("comments", "c", false, "show comments for the merge request")
	mrShowCmd.Flags().StringP("since", "s", "", "show comments since specified date (format: 2020-08-21 14:57:46.808 +0000 UTC)")
	mrShowCmd.Flags().BoolVarP(&mrShowPatch, "patch", "p", false, "show MR patches")
	mrShowCmd.Flags().BoolVarP(&mrShowPatchReverse, "reverse", "", false, "reverse order when showing MR patches (chronological instead of anti-chronological)")
	mrShowCmd.Flags().BoolVarP(&mrShowNoColorDiff, "no-color-diff", "", false, "show color diffs in comments")
	mrCmd.AddCommand(mrShowCmd)
	carapace.Gen(mrShowCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
