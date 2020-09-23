package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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

var issueNoteCmd = &cobra.Command{
	Use:              "note [remote] <id>",
	Aliases:          []string{"comment"},
	Short:            "Add a note or comment to an issue on GitLab",
	Long:             ``,
	Args:             cobra.MinimumNArgs(1),
	PersistentPreRun: LabPersistentPreRun,
	Run:              NoteRunFn,
}

func NoteRunFn(cmd *cobra.Command, args []string) {
	rn, idNum, err := parseArgs(args)
	if err != nil {
		log.Fatal(err)
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

	CreateNote(os.Args, rn, int(idNum), msgs, filename, linebreak)
}

var is_mr bool = false

func CreateNote(args []string, rn string, idNum int, msgs []string, filename string, linebreak bool) {

	var err error

	if args[1] == "mr" {
		is_mr = true
	}

	body := ""
	if filename != "" {
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		body = string(content)
	} else {
		body, err = noteMsg(msgs)
		if err != nil {
			_, f, l, _ := runtime.Caller(0)
			log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
		}
	}

	if body == "" {
		log.Fatal("aborting note due to empty note msg")
	}

	if linebreak {
		body = textToMarkdown(body)
	}

	var (
		noteURL string
	)

	if is_mr {
		noteURL, err = lab.MRCreateNote(rn, idNum, &gitlab.CreateMergeRequestNoteOptions{
			Body: &body,
		})
	} else {
		noteURL, err = lab.IssueCreateNote(rn, idNum, &gitlab.CreateIssueNoteOptions{
			Body: &body,
		})
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(noteURL)
}

func noteMsg(msgs []string) (string, error) {
	if len(msgs) > 0 {
		return strings.Join(msgs[0:], "\n\n"), nil
	}

	text, err := noteText()
	if err != nil {
		return "", err
	}

	if is_mr {
		return git.EditFile("MR_NOTE", text)
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
	issueNoteCmd.Flags().StringArrayP("message", "m", []string{}, "Use the given <msg>; multiple -m are concatenated as separate paragraphs")
	issueNoteCmd.Flags().StringP("file", "F", "", "Use the given file as the message")
	issueNoteCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks")
	issueCmd.AddCommand(issueNoteCmd)
	carapace.Gen(issueNoteCmd).PositionalCompletion(
		action.Remotes(),
		action.Issues(issueList),
	)
}
