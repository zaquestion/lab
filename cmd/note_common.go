package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

func noteRunFn(cmd *cobra.Command, args []string) {
	isMR := false
	if cmd.Parent().Name() == "mr" {
		isMR = true
	}

	reply, branchArgs, err := filterCommentArg(args)
	if err != nil {
		log.Fatal(err)
	}

	var (
		rn    string
		idNum int = 0
	)

	if isMR {
		s, mrNum, _ := parseArgsWithGitBranchMR(branchArgs)
		if mrNum == 0 {
			fmt.Println("Error: Cannot determine MR id.")
			os.Exit(1)
		}
		idNum = int(mrNum)
		rn = s
	} else {
		s, issueNum, _ := parseArgsRemoteAndID(branchArgs)
		if issueNum == 0 {
			fmt.Println("Error: Cannot determine issue id.")
			os.Exit(1)
		}
		idNum = int(issueNum)
		rn = s
	}

	msgs, err := cmd.Flags().GetStringArray("message")
	if err != nil {
		log.Fatal(err)
	}

	filename, err := cmd.Flags().GetString("file")
	if err != nil {
		log.Fatal(err)
	}

	linebreak, err := cmd.Flags().GetBool("force-linebreak")
	if err != nil {
		log.Fatal(err)
	}

	commit, err := cmd.Flags().GetString("commit")
	if err != nil {
		log.Fatal(err)
	}

	if reply != 0 {
		resolve, err := cmd.Flags().GetBool("resolve")
		if err != nil {
			log.Fatal(err)
		}
		// 'lab mr resolve' always overrides options
		if cmd.CalledAs() == "resolve" {
			resolve = true
		}

		quote, err := cmd.Flags().GetBool("quote")
		if err != nil {
			log.Fatal(err)
		}

		replyNote(rn, isMR, int(idNum), reply, quote, false, filename, linebreak, resolve, msgs)
		return
	}

	createNote(rn, isMR, int(idNum), msgs, filename, linebreak, commit, true)
}

func createCommitNote(rn string, mrID int, sha string, newFile string, oldFile string, linetype string, oldline int, newline int, comment string, block bool) {
	line := oldline
	if oldline == -1 {
		line = newline
	}

	if block {
		webURL, err := lab.CreateMergeRequestCommitDiscussion(rn, mrID, sha, newFile, oldFile, line, linetype, comment)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(webURL)
		return
	}

	webURL, err := lab.CreateCommitComment(rn, sha, newFile, oldFile, line, linetype, comment)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(webURL)
}

func getCommitBody(project string, commit string) (body string) {
	//body is going to be the commit diff
	ds, err := lab.GetCommitDiff(project, commit)
	if err != nil {
		fmt.Printf("    Could not get diff for commit %s.\n", commit)
		log.Fatal(err)
	}

	if len(ds) == 0 {
		log.Fatal("    No diff found for %s.", commit)
	}

	for _, d := range ds {
		body = body + fmt.Sprintf("| newfile: %s oldfile: %s\n", d.NewPath, d.OldPath)
		body = body + displayDiff(d.Diff, 0, 0, true)
	}

	return body
}

func createCommitComments(project string, mrID int, commit string, body string, block bool) {
	// Go through the body line-by-line and find lines that do not
	// begin with |.  These lines are comments that have been made
	// on the patch.  The lines that begin with | contain patch
	// tracking information (new line & old line number pairs,
	// and file information)
	scanner := bufio.NewScanner(strings.NewReader(body))
	lastDiffLine := ""
	comments := ""
	newfile := ""
	oldfile := ""
	diffCut := 1

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "| ") {
			comments += "\n" + line
			continue
		}

		if comments != "" && lastDiffLine == "" {
			log.Fatal("Cannot comment on first line of commit (context unknown).")
		}

		if comments != "" {
			linetype := ""
			oldLineNum := -1
			newLineNum := -1

			// parse lastDiffLine
			f := strings.Fields(strings.TrimSpace(lastDiffLine))

			// The output can be, for example:
			// | # # [no +/-] < context comment
			// | # #    +     < newline comment
			// | # #    -     < oldline comment
			// | #      -     < oldline comment
			// |   #    +     < newline comment

			// f[0] is always a "|"
			// f[1] will always be a number
			val1, _ := strconv.Atoi(f[1])
			val2, err := strconv.Atoi(f[2])

			if err == nil { // f[2] is a number
				if len(f) <= 3 { // f[3] does not exist
					oldLineNum = val1
					newLineNum = val2
					linetype = "context"
				} else {
					newLineNum = val2
					switch {
					case strings.HasPrefix(f[3], "+"):
						linetype = "new"
					case strings.HasPrefix(f[3], "-"):
						linetype = "old"
					default:
						linetype = "context"
					}
				}
			} else { // f[2] is not a number
				switch {
				case strings.HasPrefix(f[2], "+"):
					newLineNum = val1
					linetype = "new"
				case strings.HasPrefix(f[2], "-"):
					oldLineNum = val1
					linetype = "old"
				default:
					panic("unknown string in diff")
				}
			}

			createCommitNote(project, mrID, commit, newfile, oldfile, linetype, oldLineNum, newLineNum, comments, block)
			comments = ""
		}

		f := strings.Fields(strings.TrimSpace(line))
		if f[1] == "@@" {
			// In GitLab diff output, the leading "@" symbol indicates where
			// the metadata ends.  This is true even if passing over a digit
			// boundary (ie going from line 99 to 100).  This location can
			// be used to truncate the lines to only include the metadata.
			diffCut = strings.Index(line, "@") + 1
			lastDiffLine = ""
			continue
		}

		if f[1] == "newfile:" {
			// read filename
			f := strings.Split(line, " ")
			newfile = f[2]
			if len(f) < 5 {
				oldfile = ""
			} else {
				oldfile = f[4]
			}
			continue
		}

		if len(line) > diffCut {
			lastDiffLine = line[0:diffCut]
		} else {
			lastDiffLine = line
		}
	}
}

func noteGetState(rn string, isMR bool, idNum int) (state string) {
	if isMR {
		mr, err := lab.MRGet(rn, idNum)
		if err != nil {
			log.Fatal(err)
		}

		state = map[string]string{
			"opened": "OPEN",
			"closed": "CLOSED",
			"merged": "MERGED",
		}[mr.State]
	} else {
		issue, err := lab.IssueGet(rn, idNum)
		if err != nil {
			log.Fatal(err)
		}

		state = map[string]string{
			"opened": "OPEN",
			"closed": "CLOSED",
		}[issue.State]
	}

	return state
}

func createNote(rn string, isMR bool, idNum int, msgs []string, filename string, linebreak bool, commit string, hasNote bool) {
	// hasNote is used by action that take advantage of Gitlab 'quick-action' notes, which do not create a noteURL
	var err error

	body := ""
	if filename != "" {
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		body = string(content)
		if hasNote && len(msgs) > 0 {
			body += msgs[0]
		}
	} else {
		state := noteGetState(rn, isMR, idNum)

		if isMR && commit != "" {
			body = getCommitBody(rn, commit)
		}

		body, err = noteMsg(msgs, isMR, idNum, state, commit, body)
		if err != nil {
			_, f, l, _ := runtime.Caller(0)
			log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
		}
	}

	if body == "" {
		log.Fatal("aborting note due to empty note msg")
	}

	if linebreak && commit == "" {
		body = textToMarkdown(body)
	}

	var (
		noteURL string
	)

	if isMR {
		if commit != "" {
			createCommitComments(rn, int(idNum), commit, body, false)
		} else {
			noteURL, err = lab.MRCreateNote(rn, idNum, &gitlab.CreateMergeRequestNoteOptions{
				Body: &body,
			})
		}
	} else {
		noteURL, err = lab.IssueCreateNote(rn, idNum, &gitlab.CreateIssueNoteOptions{
			Body: &body,
		})
	}
	if err != nil {
		log.Fatal(err)
	}
	if hasNote {
		fmt.Println(noteURL)
	}
}

func noteMsg(msgs []string, isMR bool, idNum int, state string, commit string, body string) (string, error) {
	if len(msgs) > 0 {
		return strings.Join(msgs[0:], "\n\n"), nil
	}

	tmpl := noteGetTemplate(isMR, commit)
	text, err := noteText(idNum, state, commit, body, tmpl)
	if err != nil {
		return "", err
	}

	if isMR {
		return git.EditFile("MR_NOTE", text)
	}
	return git.EditFile("ISSUE_NOTE", text)
}

func noteGetTemplate(isMR bool, commit string) string {
	if !isMR {
		return heredoc.Doc(`
		{{.InitMsg}}
		{{.CommentChar}} This comment is being applied to {{.State}} Issue {{.IDnum}}.
		{{.CommentChar}} Comment lines beginning with '{{.CommentChar}}' are discarded.`)
	}
	if isMR && commit == "" {
		return heredoc.Doc(`
		{{.InitMsg}}
		{{.CommentChar}} This comment is being applied to {{.State}} Merge Request {{.IDnum}}.
		{{.CommentChar}} Comment lines beginning with '{{.CommentChar}}' are discarded.`)
	}
	return heredoc.Doc(`
		{{.InitMsg}}
		{{.CommentChar}} This comment is being applied to {{.State}} Merge Request {{.IDnum}} commit {{.Commit}}.
		{{.CommentChar}} Do not delete patch tracking lines that begin with '|'.
		{{.CommentChar}} Comment lines beginning with '{{.CommentChar}}' are discarded.`)
}

func noteText(idNum int, state string, commit string, body string, tmpl string) (string, error) {
	initMsg := body
	commentChar := git.CommentChar()

	if commit != "" {
		if len(commit) > 11 {
			commit = commit[:12]
		}
	}

	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return "", err
	}

	msg := &struct {
		InitMsg     string
		CommentChar string
		State       string
		IDnum       int
		Commit      string
	}{
		InitMsg:     initMsg,
		CommentChar: commentChar,
		State:       state,
		IDnum:       idNum,
		Commit:      commit,
	}

	var b bytes.Buffer
	err = t.Execute(&b, msg)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

func replyNote(rn string, isMR bool, idNum int, reply int, quote bool, update bool, filename string, linebreak bool, resolve bool, msgs []string) {

	var (
		discussions []*gitlab.Discussion
		err         error
		NoteURL     string
	)

	if isMR {
		discussions, err = lab.MRListDiscussions(rn, idNum)
	} else {
		discussions, err = lab.IssueListDiscussions(rn, idNum)
	}
	if err != nil {
		log.Fatal(err)
	}

	state := noteGetState(rn, isMR, idNum)

	for _, discussion := range discussions {
		for _, note := range discussion.Notes {

			if note.System {
				if note.ID == reply {
					fmt.Println("ERROR: Cannot reply to note", note.ID)
				}
				continue
			}

			if note.ID != reply {
				continue
			}

			body := ""
			if len(msgs) != 0 {
				body, err = noteMsg(msgs, isMR, idNum, state, "", body)
				if err != nil {
					_, f, l, _ := runtime.Caller(0)
					log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
				}
			} else if filename != "" {
				content, err := ioutil.ReadFile(filename)
				if err != nil {
					log.Fatal(err)
				}
				body = string(content)
			} else {
				noteBody := ""
				if quote {
					noteBody = note.Body
					noteBody = strings.Replace(noteBody, "\n", "\n>", -1)
					if !update {
						noteBody = ">" + noteBody + "\n"
					}
				}
				body, err = noteMsg([]string{}, isMR, idNum, state, "", noteBody)
				if err != nil {
					_, f, l, _ := runtime.Caller(0)
					log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
				}
			}

			if body == "" && !resolve {
				log.Fatal("aborting note due to empty note msg")
			}

			if linebreak {
				body = textToMarkdown(body)
			}

			if update {
				if isMR {
					NoteURL, err = lab.UpdateMRDiscussionNote(rn, idNum, discussion.ID, note.ID, body)
				} else {
					NoteURL, err = lab.UpdateIssueDiscussionNote(rn, idNum, discussion.ID, note.ID, body)
				}
			} else {
				if isMR {
					if body != "" {
						NoteURL, err = lab.AddMRDiscussionNote(rn, idNum, discussion.ID, body)
					}
					if resolve {
						NoteURL, err = lab.ResolveMRDiscussion(rn, idNum, discussion.ID, reply)
					}
				} else {
					NoteURL, err = lab.AddIssueDiscussionNote(rn, idNum, discussion.ID, body)
				}
			}
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(NoteURL)
		}
	}
}
