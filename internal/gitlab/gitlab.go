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

	"github.com/davecgh/go-spew/spew"
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

	if os.Getenv("DEBUG") != "" {
		log.Println("gitlab.host:", host)
		if len(_token) > 12 {
			log.Println("gitlab.token:", "************"+_token[12:])
		} else {
			log.Println("This token looks invalid due to it's length")
			log.Println("gitlab.token:", _token)
		}
		log.Println("gitlab.user:", user)

		// Test listing projects
		projects, _, err := lab.Projects.ListProjects(&gitlab.ListProjectsOptions{})
		if err != nil {
			log.Fatal("Error: ", err)
		}
		if len(projects) > 0 {
			spew.Dump(projects[0])
		}
	}
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
	if os.Getenv("DEBUG") != "" {
		log.Println("tmplFile:", tmplFile)
	}

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
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrProjectNotFound
	}
	if err != nil {
		return nil, err
	}
	if os.Getenv("DEBUG") != "" {
		spew.Dump(target)
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

// MergeRequest opens a merge request on GitLab
func MergeRequest(project string, opts *gitlab.CreateMergeRequestOptions) (string, error) {
	if os.Getenv("DEBUG") != "" {
		spew.Dump(opts)
	}

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

// ListMRs lists the MRs on a GitLab project
func ListMRs(project string, opts *gitlab.ListProjectMergeRequestsOptions) ([]*gitlab.MergeRequest, error) {
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

// IssueCreate opens a new issue on a GitLab Project
func IssueCreate(project string, opts *gitlab.CreateIssueOptions) (string, error) {
	if os.Getenv("DEBUG") != "" {
		spew.Dump(opts)
	}

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
	if os.Getenv("DEBUG") != "" {
		spew.Dump(opts)
	}

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

// BranchPushed checks if a branch exists on a GitLab Project
func BranchPushed(project, branch string) bool {
	p, err := FindProject(project)
	if err != nil {
		return false
	}

	b, _, err := lab.Branches.GetBranch(p.ID, branch)
	if err != nil {
		return false
	}
	return b != nil
}

// ProjectSnippetCreate creates a snippet in a project
func ProjectSnippetCreate(pid interface{}, opts *gitlab.CreateProjectSnippetOptions) (*gitlab.Snippet, error) {
	if os.Getenv("DEBUG") != "" {
		spew.Dump(opts)
	}
	snip, resp, err := lab.ProjectSnippets.CreateSnippet(pid, opts)
	if os.Getenv("DEBUG") != "" {
		fmt.Println(resp.Response.Status)
	}
	if err != nil {
		return nil, err
	}

	return snip, nil
}

// ProjectSnippetDelete deletes a project snippet
func ProjectSnippetDelete(pid interface{}, id int) error {
	resp, err := lab.ProjectSnippets.DeleteSnippet(pid, id)
	if os.Getenv("DEBUG") != "" {
		fmt.Println(resp.Response.Status)
	}
	return err
}

// ProjectSnippetList lists snippets on a project
func ProjectSnippetList(pid interface{}, opts *gitlab.ListProjectSnippetsOptions) ([]*gitlab.Snippet, error) {
	if os.Getenv("DEBUG") != "" {
		spew.Dump(opts)
	}
	snips, resp, err := lab.ProjectSnippets.ListSnippets(pid, opts)
	if os.Getenv("DEBUG") != "" {
		fmt.Println(resp.Response.Status)
	}
	if err != nil {
		return nil, err
	}
	return snips, nil
}

// SnippetCreate creates a personal snippet
func SnippetCreate(opts *gitlab.CreateSnippetOptions) (*gitlab.Snippet, error) {
	if os.Getenv("DEBUG") != "" {
		spew.Dump(opts)
	}
	snip, resp, err := lab.Snippets.CreateSnippet(opts)
	if os.Getenv("DEBUG") != "" {
		fmt.Println(resp.Response.Status)
	}
	if err != nil {
		return nil, err
	}

	return snip, nil
}

// SnippetDelete deletes a personal snippet
func SnippetDelete(id int) error {
	resp, err := lab.Snippets.DeleteSnippet(id)
	if os.Getenv("DEBUG") != "" {
		fmt.Println(resp.Response.Status)
	}
	return err
}

// SnippetList lists snippets on a project
func SnippetList(opts *gitlab.ListSnippetsOptions) ([]*gitlab.Snippet, error) {
	if os.Getenv("DEBUG") != "" {
		spew.Dump(opts)
	}
	snips, resp, err := lab.Snippets.ListSnippets(opts)
	if os.Getenv("DEBUG") != "" {
		fmt.Println(resp.Response.Status)
	}
	if err != nil {
		return nil, err
	}
	return snips, nil
}
