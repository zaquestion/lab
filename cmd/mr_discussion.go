package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrCreateDiscussionCmd = &cobra.Command{
	Use:     "discussion [remote] <id>",
	Short:   "Start a discussion on an MR on GitLab",
	Aliases: []string{"block", "thread"},
	Example: heredoc.Doc(`
		lab mr discussion origin
		lab mr discussion my_remote -m "discussion comment"
		lab mr discussion upstream -F test_file.txt
		lab mr discussion --commit abcdef123456`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, mrNum, err := parseArgsWithGitBranchMR(args)
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

		commit, err := cmd.Flags().GetString("commit")
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
		} else if commit == "" {
			body, err = mrDiscussionMsg(msgs, "\n")
			if err != nil {
				_, f, l, _ := runtime.Caller(0)
				log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
			}
		} else {
			body = getCommitBody(rn, commit)
			body, err = mrDiscussionMsg(nil, body)
			if err != nil {
				_, f, l, _ := runtime.Caller(0)
				log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
			}
			createCommitComments(rn, int(mrNum), commit, body, true)
			return
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

func mrDiscussionMsg(msgs []string, body string) (string, error) {
	if len(msgs) > 0 {
		return strings.Join(msgs[0:], "\n\n"), nil
	}

	text, err := mrDiscussionText(body)
	if err != nil {
		return "", err
	}
	return git.EditFile("MR_DISCUSSION", text)
}

func mrDiscussionText(body string) (string, error) {
	tmpl := heredoc.Doc(`
		{{.InitMsg}}
		{{.CommentChar}} Write a message for this discussion. Commented lines are discarded.`)

	initMsg := body
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
	mrCreateDiscussionCmd.Flags().StringSliceP("message", "m", []string{}, "use the given <msg>; multiple -m are concatenated as separate paragraphs")
	mrCreateDiscussionCmd.Flags().StringP("file", "F", "", "use the given file as the message")
	mrCreateDiscussionCmd.Flags().StringP("commit", "c", "", "start a thread on a commit")

	mrCmd.AddCommand(mrCreateDiscussionCmd)
	carapace.Gen(mrCreateDiscussionCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
