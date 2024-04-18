package cmd

import (
	"bytes"
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
		lab mr discussion upstream -F my_comment.txt
		lab mr discussion --commit abcdef123456
		lab mr discussion my-topic-branch
		lab mr discussion origin 123
		lab mr discussion origin my-topic-branch
		lab mr discussion --commit abcdef123456 --position=main.c:+100,100
		lab mr discussion upstream 613278108 --resolve`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {

		reply, branchArgs, err := filterCommentArg(args)
		if err != nil {
			log.Fatal(err)
		}

		rn, mrNum, err := parseArgsWithGitBranchMR(branchArgs)
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

		position, err := cmd.Flags().GetString("position")
		if err != nil {
			log.Fatal(err)
		}
		resolve, err := cmd.Flags().GetBool("resolve")
		if err != nil {
			log.Fatal(err)
		}

		var currentDiscussion *gitlab.Discussion
		var discussions []*gitlab.Discussion
		var NoteURL string

		var posFile string
		var posLineType byte
		var posLineNumberNew, posLineNumberOld uint64
		if position != "" {
			colonOffset := strings.LastIndex(position, ":")
			positionUserError := "argument to --position must match <file>:[+- ]<old_line>,<new_line>"
			if colonOffset == -1 {
				log.Fatal(positionUserError + `: missing ":"`)
			}
			posFile = position[:colonOffset]
			lineTypeOffset := colonOffset + 1
			if lineTypeOffset == len(position) {
				log.Fatal(positionUserError + `: expected one of "+- ", found end of string`)
			}
			posLineType = position[lineTypeOffset]
			if bytes.IndexByte([]byte("+- "), posLineType) == -1 {
				log.Fatal(positionUserError + fmt.Sprintf(`: expected one of "+- ", found %q`, posLineType))
			}
			oldLineOffset := colonOffset + 2
			if oldLineOffset == len(position) {
				log.Fatal(positionUserError + ": missing line numbers")
			}
			commaOffset := strings.LastIndex(position, ",")
			if commaOffset == -1 || commaOffset < colonOffset {
				log.Fatal(positionUserError + `: missing "," to separate line numbers`)
			}
			posLineNumberOld, err = strconv.ParseUint(position[oldLineOffset:commaOffset], 10, 32)
			if err != nil {
				log.Fatal(positionUserError + ":error parsing <old_line>: " + err.Error())
			}
			newNumberOffset := commaOffset + 1
			posLineNumberNew, err = strconv.ParseUint(position[newNumberOffset:], 10, 32)
			if err != nil {
				log.Fatal(positionUserError + ":error parsing <new_line>: " + err.Error())
			}
		}

		state := noteGetState(rn, true, int(mrNum))

		body := ""
		if filename != "" {
			fmt.Println("1")
			content, err := ioutil.ReadFile(filename)
			if err != nil {
				log.Fatal(err)
			}
			body = string(content)
		} else if resolve {

			discussions, err = lab.MRListDiscussions(rn, int(mrNum))
			if err != nil {
				log.Fatal(err)
			}

			// Find discussion sha based on discussion
			for _, discussion := range discussions {
				if len(discussion.Notes) > 0 && discussion.Notes[0].ID == reply {
					currentDiscussion = discussion
					break
				}
			}

			NoteURL, err = lab.ResolveMRDiscussion(rn, int(mrNum), currentDiscussion, reply)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(NoteURL)
			return
		} else if position != "" || commit == "" {
			// TODO If we are commenting on a specific position in the diff, we should include some context in the template.
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

		var discussionOpts gitlab.CreateMergeRequestDiscussionOptions
		if position != "" {
			if commit == "" {
				// We currently only support "--position" when commenting on individual commits within an MR.
				// GitLab also allows to comment on the sum of all changes of an MR.
				// To do so, we'd need to fill the NotePosition below with the target branch as "base SHA" and the source branch as "head SHA".
				// However, commenting on individual commits within an MR is superior since it gives more information to the MR author.
				// Additionally, commenting on the sum of all changes is only useful when the changes come with a messy history.
				// We shouldn't encourage that  - GitLab already does ;).
				log.Fatal("--position currently requires --commit")
			}
			parentSHA, err := git.RevParse(commit + "~")
			if err != nil {
				log.Fatal(err)
			}
			// WORKAROUND For added (-) and deleted (+) lines we only need one line number parameter, but for context lines we need both. https://gitlab.com/gitlab-org/gitlab/-/issues/325161
			newLine64 := posLineNumberNew
			if posLineType == '-' {
				newLine64 = 0
			}
			newLine := int(newLine64)

			oldLine64 := posLineNumberOld
			if posLineType == '+' {
				oldLine64 = 0
			}
			oldLine := int(oldLine64)

			positionType := "text"
			discussionOpts.Position = &gitlab.PositionOptions{
				BaseSHA:      &parentSHA,
				StartSHA:     &parentSHA,
				HeadSHA:      &commit,
				PositionType: &positionType,
				NewPath:      &posFile,
				NewLine:      &newLine,
				OldPath:      &posFile,
				OldLine:      &oldLine,
			}
		}

		if body == "" {
			log.Fatal("aborting discussion due to empty discussion msg")
		}
		discussionOpts.Body = &body

		var commitID *string
		if commit != "" {
			commitID = &commit
		}
		discussionOpts.CommitID = commitID

		discussionURL, err := lab.MRCreateDiscussion(rn, int(mrNum), &discussionOpts)
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
	mrCreateDiscussionCmd.Flags().Bool("resolve", false, "mark thread resolved")

	mrCreateDiscussionCmd.Flags().StringP("position", "", "", heredoc.Doc(`
		start a thread on a specific line of the diff
		argument must be of the form <file>":"["+" | "-" | " "]<old_line>","<new_line>
		that is, the file name, followed by the line type - one of "+" (added line),
		"-" (deleted line) or a space character (context line) - followed by
		the line number in the old version of the file, a ",", and finally
		the line number in the new version of the file. If the line type is "+", then
		<old_line> is ignored. If the line type is "-", then <new_line> is ignored.

		Here's an example diff that explains how to determine the old/new line numbers:

			--- a/README.md		old	new
			+++ b/README.md
			@@ -100,3 +100,4 @@
			 pre-context line	100	100
			-deleted line		101	101
			+added line 1		101	102
			+added line 2		101	103
			 post-context line	102	104

		# Comment on "deleted line":
		lab mr discussion --commit=commit-id --position=README.md:-101,101
		# Comment on "added line 2":
		lab mr discussion --commit=commit-id --position=README.md:+101,103
		# Comment on the "post-context line":
		lab mr discussion --commit=commit-id --position=README.md:\ 102,104`))

	mrCmd.AddCommand(mrCreateDiscussionCmd)
	carapace.Gen(mrCreateDiscussionCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
