package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrCreateDiscussionCmd = &cobra.Command{
	Use:              "discussion [remote] <id>",
	Aliases:          []string{"comment"},
	Short:            "Start a discussion on an MR on GitLab",
	Long:             ``,
	Args:             cobra.MinimumNArgs(1),
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, mrNum, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		msgs, err := cmd.Flags().GetStringSlice("message")
		if err != nil {
			log.Fatal(err)
		}

		filename, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatal(err)
		}

		body := ""
		if filename != "" {
			content, err := ioutil.ReadFile(filename)
			if err != nil {
				log.Fatal(err)
			}
			body = string(content)
		} else {
			body, err = mrDiscussionMsg(msgs)
			if err != nil {
				_, f, l, _ := runtime.Caller(0)
				log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
			}
		}

		if body == "" {
			log.Fatal("aborting discussion due to empty discussion msg")
		}

		discussionURL, err := lab.MRCreateDiscussion(rn, int(mrNum), &gitlab.CreateMergeRequestDiscussionOptions{
			Body: &body,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(discussionURL)
	},
}

func mrDiscussionMsg(msgs []string) (string, error) {
	if len(msgs) > 0 {
		return strings.Join(msgs[0:], "\n\n"), nil
	}

	text, err := mrDiscussionText()
	if err != nil {
		return "", err
	}
	return git.EditFile("MR_DISCUSSION", text)
}

func mrDiscussionText() (string, error) {
	const tmpl = `{{.InitMsg}}
{{.CommentChar}} Write a message for this discussion. Commented lines are discarded.`

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
	mrCreateDiscussionCmd.Flags().StringSliceP("message", "m", []string{}, "Use the given <msg>; multiple -m are concatenated as separate paragraphs")
	mrCreateDiscussionCmd.Flags().StringP("file", "F", "", "Use the given file as the message")

	mrCmd.AddCommand(mrCreateDiscussionCmd)
	carapace.Gen(mrCreateDiscussionCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
