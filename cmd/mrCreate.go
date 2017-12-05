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
}

func runMRCreate(cmd *cobra.Command, args []string) {
	branch, err := git.CurrentBranch()
	if err != nil {
		log.Fatal(err)
	}

	sourceRemote := determineSourceRemote(branch)
	sourceProjectName, err := git.PathWithNameSpace(sourceRemote)
	if err != nil {
		log.Fatal(err)
	}

	if !lab.BranchPushed(sourceProjectName, branch) {
		log.Fatal("aborting MR, branch not present on remote: ", sourceRemote)
	}

	targetProjectName, err := git.PathWithNameSpace(forkedFromRemote)
	if err != nil {
		log.Fatal(err)
	}
	targetProject, err := lab.FindProject(targetProjectName)
	if err != nil {
		log.Fatal(err)
	}

	msg, err := mrMsg(targetBranch, branch, sourceRemote, forkedFromRemote)
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

func determineSourceRemote(branch string) string {
	// Check if the branch is being tracked
	r, err := gitconfig.Local("branch." + branch + ".remote")
	if err == nil {
		return r
	}

	// If not, check if the fork is named after the user
	_, err = gitconfig.Local("remote." + lab.User() + ".url")
	if err == nil {
		return lab.User()
	}

	// If not, default to origin
	return "origin"
}

func mrMsg(base, head, sourceRemote, forkedFromRemote string) (string, error) {
	lastCommitMsg, err := git.LastCommitMessage()
	if err != nil {
		return "", err
	}
	const tmpl = `{{if .InitMsg}}{{.InitMsg}}{{end}}

{{if .Tmpl}}{{.Tmpl}}{{end}}
{{.CommentChar}} Requesting a merge into {{.Base}} from {{.Head}}
{{.CommentChar}}
{{.CommentChar}} Write a message for this merge request. The first block
{{.CommentChar}} of text is the title and the rest is the description.{{if .CommitLogs}}
{{.CommentChar}}
{{.CommentChar}} Changes:
{{.CommentChar}}
{{.CommitLogs}}{{end}}`

	mrTmpl := lab.LoadGitLabTmpl(lab.TmplMR)

	remoteBase := fmt.Sprintf("%s/%s", forkedFromRemote, base)
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
		Tmpl        string
		CommentChar string
		Base        string
		Head        string
		CommitLogs  string
	}{
		InitMsg:     lastCommitMsg,
		Tmpl:        mrTmpl,
		CommentChar: commentChar,
		Base:        forkedFromRemote + ":" + base,
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
