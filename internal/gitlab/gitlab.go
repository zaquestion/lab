// Package gitlab is an internal wrapper for the go-gitlab package
//
// Most functions serve to expose debug logging if set and accept a project
// name string over an ID
package gitlab

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/tcnksm/go-gitconfig"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
)

var (
	ErrProjectNotFound = errors.New("gitlab project not found")
)

var (
	lab   *gitlab.Client
	host  string
	token string
	user  string
)

const defaultGitLabHost = "https://gitlab.com"

// Host exposes the GitLab scheme://hostname used to interact with the API
func Host() string {
	return host
}

// User exposes the configured GitLab user
func User() string {
	return user
}

// Init handles all of the credential setup and prompts for user input when not present. At the end it initializes a gitlab client for use throughout lab.
func Init() {
	reader := bufio.NewReader(os.Stdin)
	var err error
	host, err = gitconfig.Entire("gitlab.host")
	if err != nil {
		fmt.Printf("Enter default GitLab host (default: %s): ", defaultGitLabHost)
		host, err = reader.ReadString('\n')
		host = strings.TrimSpace(host)
		if err != nil {
			log.Fatal(err)
		}
		if host == "" {
			host = defaultGitLabHost
		}
		cmd := git.New("config", "--global", "gitlab.host", host)
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}

	}
	var errt error
	user, err = gitconfig.Entire("gitlab.user")
	token, errt = gitconfig.Entire("gitlab.token")
	if err != nil {
		fmt.Print("Enter default GitLab user: ")
		user, err = reader.ReadString('\n')
		user = strings.TrimSpace(user)
		if err != nil {
			log.Fatal(err)
		}
		if user == "" {
			log.Fatal("git config gitlab.user must be set")
		}
		cmd := git.New("config", "--global", "gitlab.user", user)
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}

		var tokenURL string
		if strings.HasSuffix(host, "/") {
			tokenURL = host + "profile/personal_access_tokens"
		} else {
			tokenURL = host + "/profile/personal_access_tokens"
		}

		// If the default user is being set this is the first time lab
		// is being run.
		if errt != nil {
			fmt.Printf("Create a token here: %s\nEnter default GitLab token (scope: api): ", tokenURL)
			byteToken, err := terminal.ReadPassword(int(syscall.Stdin))
			if err != nil {
				log.Fatal(err)
			}
			token := strings.TrimSpace(string(byteToken))

			// Its okay for the key to be empty, since you can still call public repos
			if token != "" {
				cmd := git.New("config", "--global", "gitlab.token", token)
				err = cmd.Run()
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	lab = gitlab.NewClient(nil, token)
	lab.SetBaseURL(host + "/api/v4")

	if os.Getenv("DEBUG") != "" {
		log.Println("gitlab.host:", host)
		if len(token) > 12 {
			log.Println("gitlab.token:", "************"+token[12:])
		} else {
			log.Println("This token looks invalid due to it's length")
			log.Println("gitlab.token:", token)
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
