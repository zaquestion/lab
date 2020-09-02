package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	mrShowPatch        bool
	mrShowPatchReverse bool
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

			printMRDiscussions(discussions, since, int(mrNum))
		}
	},
}

func findLocalRemote(ProjectID int) string {
	var remote string

	project, err := lab.GetProject(ProjectID)
	if err != nil {
		log.Fatal(err)
	}
	remotes_str, err := git.GetLocalRemotes()
	if err != nil {
		log.Fatal(err)
	}
	remotes := strings.Split(remotes_str, "\n")

	// find the matching local remote for this project
	for r := range remotes {
		// The fetch and push entries can be different for a remote.
		// Only the fetch entry is useful.
		if strings.Contains(remotes[r], project.SSHURLToRepo+" (fetch)") {
			found := strings.Split(remotes[r], "\t")
			remote = found[0]
			break
		}
	}

	if remote == "" {
		log.Fatal("remote for ", project.SSHURLToRepo, "not found in local remotes")
	}
	return remote
}

func printMR(mr *gitlab.MergeRequest, project string, renderMarkdown bool) {
	assignee := "None"
	milestone := "None"
	labels := "None"
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

func printMRDiscussions(discussions []*gitlab.Discussion, since string, mrNum int) {
	NewAccessTime := time.Now().UTC()

	// default path for metadata config file
	metadatafile := ".git/lab/show_metadata.hcl"

	viper.Reset()
	viper.AddConfigPath(".git/lab")
	viper.SetConfigName("show_metadata")
	viper.SetConfigType("hcl")
	// write data
	if _, ok := viper.ReadInConfig().(viper.ConfigFileNotFoundError); ok {
		if _, err := os.Stat(".git/lab"); os.IsNotExist(err) {
			os.MkdirAll(".git/lab", os.ModePerm)
		}
		if err := viper.WriteConfigAs(metadatafile); err != nil {
			log.Fatal(err)
		}
		if err := viper.ReadInConfig(); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := viper.ReadInConfig(); err != nil {
			log.Fatal(err)
		}
	}

	mrEntry := fmt.Sprintf("mr%d", mrNum)
	// if specified on command line use that, o/w use config, o/w Now
	var (
		CompareTime time.Time
		err         error
		sinceIsSet  = true
	)
	CompareTime, err = dateparse.ParseLocal(since)
	if err != nil || CompareTime.IsZero() {
		CompareTime = viper.GetTime(mrEntry)
		if CompareTime.IsZero() {
			CompareTime = time.Now().UTC()
		}
		sinceIsSet = false
	}

	// for available fields, see
	// https://godoc.org/github.com/xanzy/go-gitlab#Note
	// https://godoc.org/github.com/xanzy/go-gitlab#Discussion
	for _, discussion := range discussions {
		for i, note := range discussion.Notes {

			// skip system notes
			if note.System {
				continue
			}

			indentHeader, indentNote := "", ""
			commented := "commented"
			if !time.Time(*note.CreatedAt).Equal(time.Time(*note.UpdatedAt)) {
				commented = "updated comment"
			}

			if !discussion.IndividualNote {
				indentNote = "    "

				if i == 0 {
					commented = "started a discussion"
				} else {
					indentHeader = "    "
				}
			}

			printit := color.New().PrintfFunc()
			printit(`
%s-----------------------------------`, indentHeader)

			if time.Time(*note.UpdatedAt).After(CompareTime) {
				printit = color.New(color.Bold).PrintfFunc()
			}
			printit(`
%s%s %s at %s

%s%s
`,
				indentHeader, note.Author.Username, commented, time.Time(*note.UpdatedAt).String(),
				indentNote, note.Body)
		}
	}

	if sinceIsSet == false {
		viper.Set(mrEntry, NewAccessTime)
		viper.WriteConfigAs(metadatafile)
	}
}

func init() {
	mrShowCmd.Flags().BoolP("no-markdown", "M", false, "Don't use markdown renderer to print the issue description")
	mrShowCmd.Flags().BoolP("comments", "c", false, "Show comments for the merge request")
	mrShowCmd.Flags().StringP("since", "s", "", "Show comments since specified date (format: 2020-08-21 14:57:46.808 +0000 UTC)")
	mrShowCmd.Flags().BoolVarP(&mrShowPatch, "patch", "p", false, "Show MR patches")
	mrShowCmd.Flags().BoolVarP(&mrShowPatchReverse, "reverse", "", false, "Reverse order when showing MR patches (chronological instead of anti-chronological)")
	mrCmd.AddCommand(mrShowCmd)
	carapace.Gen(mrShowCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
