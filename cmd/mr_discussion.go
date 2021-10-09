package cmd

import (
	"fmt"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrCreateDiscussionCmd = &cobra.Command{
	Use:     "discussion [remote] [<MR ID or branch>]",
	Short:   "Start a discussion on an MR on GitLab",
	Aliases: []string{"block", "thread"},
	Example: heredoc.Doc(`
		lab mr discussion
		lab mr discussion origin
		lab mr discussion my_remote -m "discussion comment"
		lab mr discussion upstream -F test_file.txt
		lab mr discussion --commit abcdef123456
		lab mr discussion my-topic-branch
		lab mr discussion origin 123
		lab mr discussion origin my-topic-branch
		`),
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

		state := noteGetState(rn, true, int(mrNum))

		body := ""
		if filename != "" {
			content, err := ioutil.ReadFile(filename)
			if err != nil {
				log.Fatal(err)
			}
			body = string(content)
		} else if commit == "" {
			body, err = mrDiscussionMsg(int(mrNum), state, commit, msgs, "\n")
			if err != nil {
				_, f, l, _ := runtime.Caller(0)
				log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
			}
		} else {
			body = getCommitBody(rn, commit)
			body, err = mrDiscussionMsg(int(mrNum), state, commit, nil, body)
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

func mrDiscussionMsg(mrNum int, state string, commit string, msgs []string, body string) (string, error) {
	if len(msgs) > 0 {
		return strings.Join(msgs[0:], "\n\n"), nil
	}

	tmpl := mrDiscussionGetTemplate(commit)
	text, err := noteText(mrNum, state, commit, body, tmpl)
	if err != nil {
		return "", err
	}
	return git.EditFile("MR_DISCUSSION", text)
}

func mrDiscussionGetTemplate(commit string) string {
	if commit == "" {
		return heredoc.Doc(`
		{{.InitMsg}}
		{{.CommentChar}} This thread is being started on {{.State}} Merge Request {{.IDnum}}.
		{{.CommentChar}} Comment lines beginning with '{{.CommentChar}}' are discarded.`)
	}
	return heredoc.Doc(`
		{{.InitMsg}}
		{{.CommentChar}} This thread is being started on {{.State}} Merge Request {{.IDnum}} commit {{.Commit}}.
		{{.CommentChar}} Do not delete patch tracking lines that begin with '|'.
		{{.CommentChar}} Comment lines beginning with '{{.CommentChar}}' are discarded.`)
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
