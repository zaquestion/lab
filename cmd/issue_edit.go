package cmd

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var issueEditCmd = &cobra.Command{
	Use:     "edit [remote] <id>",
	Aliases: []string{"update"},
	Short:   "Edit or update an issue",
	Long:    ``,
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// get remote and issue from cmd arguments
		rn, issueNum, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		// get existing issue
		issue, err := lab.IssueGet(rn, int(issueNum))
		if err != nil {
			log.Fatal(err)
		}

		//
		// Labels
		//
		// get the labels to add
		labels, err := cmd.Flags().GetStringSlice("label")
		if err != nil {
			log.Fatal(err)
		}

		// get the labels to remove
		unlabels, err := cmd.Flags().GetStringSlice("unlabel")
		if err != nil {
			log.Fatal(err)
		}

		// prepare the new list of labels, adding & removing as needed
		labels = difference(union(issue.Labels, labels), unlabels)

		//
		// Title & Body
		//
		title, body := issue.Title, issue.Description

		// get all of the "message" flags
		msgs, err := cmd.Flags().GetStringSlice("message")
		if err != nil {
			log.Fatal(err)
		}

		// if "title" was passed, prepend it to msgs
		t, err := cmd.Flags().GetString("title")
		if err != nil {
			log.Fatal(err)
		}
		if t != "" {
			msgs = append([]string{t}, msgs...)
		}

		openEditor, err := cmd.Flags().GetBool("edit")
		if err != nil {
			log.Fatal(err)
		}

		if len(msgs) != 0 || cmd.Flags().NFlag() == 0 || openEditor {
			// given the old title, description and parameters, get the "new" title
			// and description
			title, body, err = issueEditMsg(title, body, msgs)
			if err != nil {
				_, f, l, _ := runtime.Caller(0)
				log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
			}
			if title == "" {
				log.Fatal("aborting update due to empty issue title")
			}
			if title == issue.Title && body == issue.Description {
				log.Fatal("aborting update because the title and description haven't changed")
			}
		}

		opts := &gitlab.UpdateIssueOptions{
			Title:       &title,
			Description: &body,
		}

		if !same(issue.Labels, labels) {
			opts.Labels = gitlab.Labels(labels)
		}

		// do the update
		issueURL, err := lab.IssueUpdate(rn, int(issueNum), opts)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(issueURL)
	},
}

func issueEditMsg(title string, body string, msgs []string) (string, string, error) {
	// if only 1 arg was given, just update the title
	// if >1 args given, we have a title and can build a description
	if len(msgs) == 1 {
		return msgs[0], body, nil
	} else if len(msgs) > 1 {
		return msgs[0], strings.Join(msgs[1:], "\n\n"), nil
	}

	// no args given, so kick it out to the users editor
	text, err := issueEditText(title, body)
	if err != nil {
		return "", "", err
	}
	return git.Edit("ISSUE_EDIT", text)
}

func issueEditText(title string, body string) (string, error) {
	const tmpl = `{{.InitMsg}}

{{.CommentChar}} Edit the title and/or description of this issue. The first
{{.CommentChar}} block of text is the title and the rest is the description.`

	msg := &struct {
		InitMsg     string
		CommentChar string
	}{
		InitMsg:     title + "\n\n" + body,
		CommentChar: git.CommentChar(),
	}

	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	err = t.Execute(&b, msg)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

// union returns all the unique elements in a and b
func union(a, b []string) []string {
	mb := map[string]bool{}
	ab := []string{}
	for _, x := range b {
		mb[x] = true
		// add all of b's elements to ab
		ab = append(ab, x)
	}
	for _, x := range a {
		if _, ok := mb[x]; !ok {
			// if a's elements aren't in b, add them to ab
			// if they are, we don't need to add them
			ab = append(ab, x)
		}
	}
	return ab
}

// difference returns the elements in a that aren't in b
func difference(a, b []string) []string {
	mb := map[string]bool{}
	for _, x := range b {
		mb[x] = true
	}
	ab := []string{}
	for _, x := range a {
		if _, ok := mb[x]; !ok {
			ab = append(ab, x)
		}
	}
	return ab
}

// same returns true if a and b contain the same strings
func same(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	mb := map[string]bool{}
	for _, x := range b {
		mb[x] = true
	}

	for _, x := range a {
		if _, ok := mb[x]; !ok {
			return false
		}
	}
	return true
}

func init() {
	issueEditCmd.Flags().StringP("title", "t", "", "Set the issue title")
	issueEditCmd.Flags().StringSliceP("message", "m", []string{}, "Use the given <msg>; multiple -m are concatenated as separate paragraphs")
	issueEditCmd.Flags().StringSliceP("label", "l", []string{}, "Add the given label(s) to the issue")
	issueEditCmd.Flags().StringSliceP("unlabel", "L", []string{}, "Remove the given label(s) from the issue")
	issueEditCmd.Flags().Bool("edit", false, "Open the issue in an editor (default if no other flags given)")
	issueEditCmd.Flags().StringSliceP("assign", "a", []string{}, "Add an assignee by username")
	issueEditCmd.Flags().StringSliceP("unassign", "A", []string{}, "Remove an assigne by username")
	// issueEditCmd.Flags().SortFlags = false
	issueCmd.AddCommand(issueEditCmd)
}
