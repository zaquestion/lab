package cmd

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/tcnksm/go-gitconfig"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// Only supporting merges into master currently, using this constant to keep
// track of reference when setting your own base allowed
const (
	targetBranch = "master"
)

var (
	// Will be updated to upstream in init() if user if remote exists
	targetRemote = "origin"
)

// mrCmd represents the mr command
var mrCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Open a merge request on GitLab",
	Long:  `Currently only supports MRs into master`,
	Args:  cobra.ExactArgs(0),
	Run:   runMRCreate,
}

func init() {
	mrCmd.AddCommand(mrCreateCmd)
	_, err := gitconfig.Local("remote.upstream.url")
	if err == nil {
		targetRemote = "upstream"
	}
}

func runMRCreate(cmd *cobra.Command, args []string) {
	branch, err := git.CurrentBranch()
	if err != nil {
		log.Fatal(err)
	}

	sourceRemote, err := gitconfig.Local("branch." + branch + ".remote")
	if err != nil {
		sourceRemote = "origin"
	}
	sourceProjectName, err := git.PathWithNameSpace(sourceRemote)
	if err != nil {
		log.Fatal(err)
	}

	targetProjectName, err := git.PathWithNameSpace(targetRemote)
	if err != nil {
		log.Fatal(err)
	}
	targetProject, err := lab.FindProject(targetProjectName)
	if err != nil {
		log.Fatal(err)
	}

	msg, err := mrMsg(targetBranch, branch, sourceRemote, targetRemote)
	if err != nil {
		log.Fatal(err)
	}

	title, body, err := git.Edit("MERGEREQ", msg)
	if err != nil {
		_, f, l, _ := runtime.Caller(0)
		log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
	}

	if title == "" {
		log.Fatal("aborting MR due to empty MR msg")
	}

	mrURL, err := lab.MergeRequest(sourceProjectName, &gitlab.CreateMergeRequestOptions{
		SourceBranch:    &branch,
		TargetBranch:    gitlab.String(targetBranch),
		TargetProjectID: &targetProject.ID,
		Title:           &title,
		Description:     &body,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(mrURL + "/diffs")
}

func mrMsg(base, head, sourceRemote, targetRemote string) (string, error) {
	lastCommitMsg, err := git.LastCommitMessage()
	if err != nil {
		return "", err
	}
	const tmpl = `{{if .InitMsg}}{{.InitMsg}}
{{end}}
{{.CommentChar}} Requesting a merge into {{.Base}} from {{.Head}}
{{.CommentChar}}
{{.CommentChar}} Write a message for this merge request. The first block
{{.CommentChar}} of text is the title and the rest is the description.{{if .CommitLogs}}
{{.CommentChar}}
{{.CommentChar}} Changes:
{{.CommentChar}}
{{.CommitLogs}}{{end}}`

	remoteBase := fmt.Sprintf("%s/%s", targetRemote, base)
	commitLogs, err := git.Log(remoteBase, head)
	if err != nil {
		return "", err
	}
	startRegexp := regexp.MustCompilePOSIX("^")
	commentChar := git.CommentChar()
	commitLogs = strings.TrimSpace(commitLogs)
	commitLogs = startRegexp.ReplaceAllString(commitLogs, fmt.Sprintf("%s ", commentChar))

	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return "", err
	}

	msg := &struct {
		InitMsg     string
		CommentChar string
		Base        string
		Head        string
		CommitLogs  string
	}{
		InitMsg:     lastCommitMsg,
		CommentChar: commentChar,
		Base:        targetRemote + ":" + base,
		Head:        sourceRemote + ":" + head,
		CommitLogs:  commitLogs,
	}

	var b bytes.Buffer
	err = t.Execute(&b, msg)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}
