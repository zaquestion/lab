package cmd

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var issueCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Open an issue on GitLab",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		rn, err := git.PathWithNameSpace(forkedFromRemote)
		if err != nil {
			log.Fatal(err)
		}

		msg, err := issueMsg()
		if err != nil {
			log.Fatal(err)
		}

		title, body, err := git.Edit("ISSUE", msg)
		if err != nil {
			_, f, l, _ := runtime.Caller(0)
			log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
		}
		if title == "" {
			log.Fatal("aborting issue due to empty issue msg")
		}

		issueURL, err := lab.IssueCreate(rn, &gitlab.CreateIssueOptions{
			Title:       &title,
			Description: &body,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(issueURL)
	},
}

func issueMsg() (string, error) {
	const tmpl = `{{.InitMsg}}
{{.CommentChar}} Write a message for this issue. The first block
{{.CommentChar}} of text is the title and the rest is the description.`

	issueTmpl := lab.LoadGitLabTmpl(lab.TmplIssue)

	initMsg := "\n"
	if issueTmpl != "" {
		initMsg = "\n\n" + issueTmpl
	}

	commentChar := git.CommentChar()

	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return "", err
	}

	msg := &struct {
		InitMsg     string
		CommentChar string
	}{
		InitMsg:     initMsg,
		CommentChar: commentChar,
	}

	var b bytes.Buffer
	err = t.Execute(&b, msg)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

func init() {
	issueCmd.AddCommand(issueCreateCmd)
}
