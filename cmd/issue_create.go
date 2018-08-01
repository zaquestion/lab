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

var issueCreateCmd = &cobra.Command{
	Use:   "create [remote]",
	Short: "Open an issue on GitLab",
	Long:  ``,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		msgs, err := cmd.Flags().GetStringSlice("message")
		if err != nil {
			log.Fatal(err)
		}
		assignees, err := cmd.Flags().GetStringSlice("assignees")
		if err != nil {
			log.Fatal(err)
		}
		labels, err := cmd.Flags().GetStringSlice("label")
		if err != nil {
			log.Fatal(err)
		}
		remote := forkedFromRemote
		if len(args) > 0 {
			ok, err := git.IsRemote(args[0])
			if err != nil {
				log.Fatal(err)
			}
			if ok {
				remote = args[0]
			}
		}
		rn, err := git.PathWithNameSpace(remote)
		if err != nil {
			log.Fatal(err)
		}

		title, body, err := issueMsg(msgs)
		if err != nil {
			_, f, l, _ := runtime.Caller(0)
			log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
		}
		if title == "" {
			log.Fatal("aborting issue due to empty issue msg")
		}

		assigneeIDs := make([]int, len(assignees))
		for i, a := range assignees {
			assigneeIDs[i] = *getAssigneeID(a)
		}

		issueURL, err := lab.IssueCreate(rn, &gitlab.CreateIssueOptions{
			Title:       &title,
			Description: &body,
			Labels:      gitlab.Labels(labels),
			AssigneeIDs: assigneeIDs,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(issueURL)
	},
}

func issueMsg(msgs []string) (string, string, error) {
	if len(msgs) > 0 {
		return msgs[0], strings.Join(msgs[1:], "\n\n"), nil
	}

	text, err := issueText()
	if err != nil {
		return "", "", err
	}
	return git.Edit("ISSUE", text)
}

func issueText() (string, error) {
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
	issueCreateCmd.Flags().StringSliceP("message", "m", []string{}, "Use the given <msg>; multiple -m are concatenated as separate paragraphs")
	issueCreateCmd.Flags().StringSliceP("label", "l", []string{}, "Set the given label(s) on the created issue")
	issueCreateCmd.Flags().StringSliceP("assignees", "a", []string{}, "Set assignees by username")
	issueCmd.AddCommand(issueCreateCmd)
}
