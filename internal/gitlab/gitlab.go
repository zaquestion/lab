// Package gitlab is an internal wrapper for the go-gitlab package
//
// Most functions serve to expose debug logging if set and accept a project
// name string over an ID
package gitlab

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
)

var (
	ErrProjectNotFound = errors.New("gitlab project not found")
)

var (
	lab  *gitlab.Client
	host string
	user string
)

// Host exposes the GitLab scheme://hostname used to interact with the API
func Host() string {
	return host
}

// User exposes the configured GitLab user
func User() string {
	return user
}

// Init initializes a gitlab client for use throughout lab.
func Init(_host, _user, _token string) {
	host = _host
	user = _user
	lab = gitlab.NewClient(nil, _token)
	lab.SetBaseURL(host + "/api/v4")
}

// Defines filepath for default GitLab templates
const (
	TmplMR    = "merge_request_templates/default.md"
	TmplIssue = "issue_templates/default.md"
)

// LoadGitLabTmpl loads gitlab templates for use in creating Issues and MRs
//
// https://gitlab.com/help/user/project/description_templates.md#setting-a-default-template-for-issues-and-merge-requests
func LoadGitLabTmpl(tmplName string) string {
	wd, err := git.WorkingDir()
	if err != nil {
		log.Fatal(err)
	}

	tmplFile := filepath.Join(wd, ".gitlab", tmplName)
	f, err := os.Open(tmplFile)
	if os.IsNotExist(err) {
		return ""
	} else if err != nil {
		log.Fatal(err)
	}

	tmpl, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	return string(tmpl[:len(tmpl)-1])
}

var (
	localProjects map[string]*gitlab.Project = make(map[string]*gitlab.Project)
)

// GetProject looks up a Gitlab project by ID.
func GetProject(projectID int) (*gitlab.Project, error) {
	target, resp, err := lab.Projects.GetProject(projectID)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrProjectNotFound
	}
	if err != nil {
		return nil, err
	}
	return target, nil
}

// FindProject looks up the Gitlab project. If the namespace is not provided in
// the project string it will search for projects in the users namespace
func FindProject(project string) (*gitlab.Project, error) {
	if target, ok := localProjects[project]; ok {
		return target, nil
	}

	search := project
	// Assuming that a "/" in the project means its owned by an org
	if !strings.Contains(project, "/") {
		search = user + "/" + project
	}

	target, resp, err := lab.Projects.GetProject(search)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrProjectNotFound
	}
	if err != nil {
		return nil, err
	}
	// fwiw, I feel bad about this
	localProjects[project] = target

	return target, nil
}

// Fork creates a user fork of a GitLab project
func Fork(project string) (string, error) {
	if !strings.Contains(project, "/") {
		return "", errors.New("remote must include namespace")
	}
	parts := strings.Split(project, "/")

	// See if a fork already exists
	target, err := FindProject(parts[1])
	if err == nil {
		return target.SSHURLToRepo, nil
	} else if err != nil && err != ErrProjectNotFound {
		return "", err
	}

	target, err = FindProject(project)
	if err != nil {
		return "", err
	}

	fork, _, err := lab.Projects.ForkProject(target.ID)
	if err != nil {
		return "", err
	}

	return fork.SSHURLToRepo, nil
}

// MRCreate opens a merge request on GitLab
func MRCreate(project string, opts *gitlab.CreateMergeRequestOptions) (string, error) {
	p, err := FindProject(project)
	if err != nil {
		return "", err
	}

	mr, _, err := lab.MergeRequests.CreateMergeRequest(p.ID, opts)
	if err != nil {
		return "", err
	}
	return mr.WebURL, nil
}

// MRGet retrieves the merge request from GitLab project
func MRGet(project string, mrNum int) (*gitlab.MergeRequest, error) {
	p, err := FindProject(project)
	if err != nil {
		return nil, err
	}

	mr, _, err := lab.MergeRequests.GetMergeRequest(p.ID, mrNum)
	if err != nil {
		return nil, err
	}

	return mr, nil
}

// MRList lists the MRs on a GitLab project
func MRList(project string, opts *gitlab.ListProjectMergeRequestsOptions) ([]*gitlab.MergeRequest, error) {
	p, err := FindProject(project)
	if err != nil {
		return nil, err
	}

	list, _, err := lab.MergeRequests.ListProjectMergeRequests(p.ID, opts)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// MRClose closes an mr on a GitLab project
func MRClose(pid interface{}, id int) error {
	mr, _, err := lab.MergeRequests.GetMergeRequest(pid, id)
	if err != nil {
		return err
	}
	if mr.State == "closed" {
		return fmt.Errorf("mr already closed")
	}
	_, _, err = lab.MergeRequests.UpdateMergeRequest(pid, int(id), &gitlab.UpdateMergeRequestOptions{
		StateEvent: gitlab.String("close"),
	})
	if err != nil {
		return err
	}
	return nil
}

// MRMerge merges an mr on a GitLab project
func MRMerge(pid interface{}, id int) error {
	_, _, err := lab.MergeRequests.AcceptMergeRequest(pid, int(id), &gitlab.AcceptMergeRequestOptions{
		MergeWhenPipelineSucceeds: gitlab.Bool(true),
	})
	if err != nil {
		return err
	}
	return nil
}

// IssueCreate opens a new issue on a GitLab Project
func IssueCreate(project string, opts *gitlab.CreateIssueOptions) (string, error) {
	p, err := FindProject(project)
	if err != nil {
		return "", err
	}

	mr, _, err := lab.Issues.CreateIssue(p.ID, opts)
	if err != nil {
		return "", err
	}
	return mr.WebURL, nil
}

// IssueGet retrieves the issue information from a GitLab project
func IssueGet(project string, issueNum int) (*gitlab.Issue, error) {
	p, err := FindProject(project)
	if err != nil {
		return nil, err
	}

	issue, _, err := lab.Issues.GetIssue(p.ID, issueNum)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

// IssueList gets a list of issues on a GitLab Project
func IssueList(project string, opts *gitlab.ListProjectIssuesOptions) ([]*gitlab.Issue, error) {
	p, err := FindProject(project)
	if err != nil {
		return nil, err
	}

	list, _, err := lab.Issues.ListProjectIssues(p.ID, opts)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// IssueClose closes an issue on a GitLab project
func IssueClose(pid interface{}, id int) error {
	_, err := lab.Issues.DeleteIssue(pid, int(id))
	if err != nil {
		return err
	}
	return nil
}

// BranchPushed checks if a branch exists on a GitLab project
func BranchPushed(pid interface{}, branch string) bool {
	b, _, err := lab.Branches.GetBranch(pid, branch)
	if err != nil {
		return false
	}
	return b != nil
}

// ProjectSnippetCreate creates a snippet in a project
func ProjectSnippetCreate(pid interface{}, opts *gitlab.CreateProjectSnippetOptions) (*gitlab.Snippet, error) {
	snip, _, err := lab.ProjectSnippets.CreateSnippet(pid, opts)
	if err != nil {
		return nil, err
	}

	return snip, nil
}

// ProjectSnippetDelete deletes a project snippet
func ProjectSnippetDelete(pid interface{}, id int) error {
	_, err := lab.ProjectSnippets.DeleteSnippet(pid, id)
	return err
}

// ProjectSnippetList lists snippets on a project
func ProjectSnippetList(pid interface{}, opts *gitlab.ListProjectSnippetsOptions) ([]*gitlab.Snippet, error) {
	snips, _, err := lab.ProjectSnippets.ListSnippets(pid, opts)
	if err != nil {
		return nil, err
	}
	return snips, nil
}

// SnippetCreate creates a personal snippet
func SnippetCreate(opts *gitlab.CreateSnippetOptions) (*gitlab.Snippet, error) {
	snip, _, err := lab.Snippets.CreateSnippet(opts)
	if err != nil {
		return nil, err
	}

	return snip, nil
}

// SnippetDelete deletes a personal snippet
func SnippetDelete(id int) error {
	_, err := lab.Snippets.DeleteSnippet(id)
	return err
}

// SnippetList lists snippets on a project
func SnippetList(opts *gitlab.ListSnippetsOptions) ([]*gitlab.Snippet, error) {
	snips, _, err := lab.Snippets.ListSnippets(opts)
	if err != nil {
		return nil, err
	}
	return snips, nil
}

// Lint validates .gitlab-ci.yml contents
func Lint(content string) (bool, error) {
	lint, _, err := lab.Validate.Lint(content)
	if err != nil {
		return false, err
	}
	if len(lint.Errors) > 0 {
		return false, errors.New(strings.Join(lint.Errors, " - "))
	}
	return lint.Status == "valid", nil
}
