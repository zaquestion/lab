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
	zsh "github.com/rsteube/cobra-zsh-gen"
)

var issueCreateNoteCmd = &cobra.Command{
	Use:     "note [remote] <id>",
	Aliases: []string{"comment"},
	Short:   "Add a note or comment to an issue on GitLab",
	Long:    ``,
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rn, issueNum, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		msgs, err := cmd.Flags().GetStringSlice("message")
		if err != nil {
			log.Fatal(err)
		}

		body, err := noteMsg(msgs)
		if err != nil {
			_, f, l, _ := runtime.Caller(0)
			log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
		}
		if body == "" {
			log.Fatal("aborting note due to empty note msg")
		}

		noteURL, err := lab.IssueCreateNote(rn, int(issueNum), &gitlab.CreateIssueNoteOptions{
			Body: &body,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(noteURL)
	},
}

//
func noteMsg(msgs []string) (string, error) {
	if len(msgs) > 0 {
		return strings.Join(msgs[0:], "\n\n"), nil
	}

	text, err := noteText()
	if err != nil {
		return "", err
	}
	return git.EditFile("ISSUE_NOTE", text)
}

func noteText() (string, error) {
	const tmpl = `{{.InitMsg}}
{{.CommentChar}} Write a message for this note. Commented lines are discarded.`

	initMsg := "\n"
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
	issueCreateNoteCmd.Flags().StringSliceP("message", "m", []string{}, "Use the given <msg>; multiple -m are concatenated as separate paragraphs")

	zsh.Wrap(issueCreateNoteCmd).MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	zsh.Wrap(issueCreateNoteCmd).MarkZshCompPositionalArgumentCustom(2, "__lab_completion_issue $words[2]")
	issueCmd.AddCommand(issueCreateNoteCmd)
}
