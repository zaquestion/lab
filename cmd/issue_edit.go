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
	Short:   "Update title and/or description of an issue",
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

		// given the old title, description and parameters, get the "new" title
		// and description
		title, body, err := issueEditMsg(issue.Title, issue.Description, msgs)
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

		// do the update
		issueURL, err := lab.IssueUpdate(rn, int(issueNum), &gitlab.UpdateIssueOptions{
			Title:       &title,
			Description: &body,
		})
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
	text, err := issueUpdateText(title, body)
	if err != nil {
		return "", "", err
	}
	return git.Edit("ISSUE_EDIT", text)
}

func issueUpdateText(title string, body string) (string, error) {
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

func init() {
	issueEditCmd.Flags().StringSliceP("message", "m", []string{}, "Use the given <msg>; multiple -m are concatenated as separate paragraphs")
	issueEditCmd.Flags().StringP("title", "t", "", "Set the issue title")
	issueCmd.AddCommand(issueEditCmd)
}
