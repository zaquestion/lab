// Package gitlab is an internal wrapper for the go-gitlab package
//
// Most functions serve to expose debug logging if set and accept a project
// name string over an ID.
package gitlab

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
)

var (
	// ErrProjectNotFound is returned when a GitLab project cannot be found.
	ErrProjectNotFound = errors.New("gitlab project not found, verify you have access to the requested resource")
)

var (
	lab   *gitlab.Client
	host  string
	user  string
	token string
)

// Host exposes the GitLab scheme://hostname used to interact with the API
func Host() string {
	return host
}

// User exposes the configured GitLab user
func User() string {
	return user
}

func UserID() (int, error) {
	u, _, err := lab.Users.CurrentUser()
	if err != nil {
		return 0, err
	}
	return u.ID, nil
}

func UserIDByUserName(username string) (int, error) {
	opts := gitlab.ListUsersOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 1,
		},
		Username: &username,
	}
	users, _, err := lab.Users.ListUsers(&opts)
	if err != nil {
		return 0, err
	}
	for _, user := range users {
		return user.ID, nil
	}

	return 0, errors.New("No user found with username " + username)
}

// Init initializes a gitlab client for use throughout lab.
func Init(_host, _user, _token string, allowInsecure bool) {
	if len(_host) > 0 && _host[len(_host)-1:][0] == '/' {
		_host = _host[0 : len(_host)-1]
	}
	host = _host
	user = _user
	token = _token

	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: allowInsecure,
			},
		},
	}

	lab, _ = gitlab.NewClient(token, gitlab.WithHTTPClient(httpClient), gitlab.WithBaseURL(host+"/api/v4"))
}

func InitWithCustomCA(_host, _user, _token, caFile string) error {
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		return err
	}
	// use system cert pool as a baseline
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		return err
	}
	caCertPool.AppendCertsFromPEM(caCert)

	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	lab, _ = gitlab.NewClient(token, gitlab.WithHTTPClient(httpClient), gitlab.WithBaseURL(host+"/api/v4"))
	return nil
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

	return strings.TrimSpace(string(tmpl))
}

var (
	localProjects map[string]*gitlab.Project = make(map[string]*gitlab.Project)
)

// GetProject looks up a Gitlab project by ID.
func GetProject(projectID interface{}) (*gitlab.Project, error) {
	target, resp, err := lab.Projects.GetProject(projectID, nil)
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

	target, resp, err := lab.Projects.GetProject(search, nil)
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

	fork, _, err := lab.Projects.ForkProject(target.ID, nil)
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

// MRCreateDiscussion creates a discussion on a merge request on GitLab
func MRCreateDiscussion(project string, mrNum int, opts *gitlab.CreateMergeRequestDiscussionOptions) (string, error) {
	p, err := FindProject(project)
	if err != nil {
		return "", err
	}

	discussion, _, err := lab.Discussions.CreateMergeRequestDiscussion(p.ID, mrNum, opts)
	if err != nil {
		return "", err
	}

	// Unlike MR, Note has no WebURL property, so we have to create it
	// ourselves from the project, noteable id and note id
	note := discussion.Notes[0]
	return fmt.Sprintf("%s/merge_requests/%d#note_%d", p.WebURL, note.NoteableIID, note.ID), nil
}

// MRUpdate edits an merge request on a GitLab project
func MRUpdate(project string, mrNum int, opts *gitlab.UpdateMergeRequestOptions) (string, error) {
	p, err := FindProject(project)
	if err != nil {
		return "", err
	}

	issue, _, err := lab.MergeRequests.UpdateMergeRequest(p.ID, mrNum, opts)
	if err != nil {
		return "", err
	}
	return issue.WebURL, nil
}

// MRCreateNote adds a note to a merge request on GitLab
func MRCreateNote(project string, mrNum int, opts *gitlab.CreateMergeRequestNoteOptions) (string, error) {
	p, err := FindProject(project)
	if err != nil {
		return "", err
	}

	note, _, err := lab.Notes.CreateMergeRequestNote(p.ID, mrNum, opts)
	if err != nil {
		return "", err
	}
	// Unlike MR, Note has no WebURL property, so we have to create it
	// ourselves from the project, noteable id and note id
	return fmt.Sprintf("%s/merge_requests/%d#note_%d", p.WebURL, note.NoteableIID, note.ID), nil
}

// MRGet retrieves the merge request from GitLab project
func MRGet(project string, mrNum int) (*gitlab.MergeRequest, error) {
	p, err := FindProject(project)
	if err != nil {
		return nil, err
	}

	mr, _, err := lab.MergeRequests.GetMergeRequest(p.ID, mrNum, nil)
	if err != nil {
		return nil, err
	}

	return mr, nil
}

// MRList lists the MRs on a GitLab project
func MRList(project string, opts gitlab.ListProjectMergeRequestsOptions, n int) ([]*gitlab.MergeRequest, error) {
	if n == -1 {
		opts.PerPage = 100
	}
	p, err := FindProject(project)
	if err != nil {
		return nil, err
	}

	list, resp, err := lab.MergeRequests.ListProjectMergeRequests(p.ID, &opts)
	if err != nil {
		return nil, err
	}
	if resp.CurrentPage == resp.TotalPages {
		return list, nil
	}
	opts.Page = resp.NextPage
	for len(list) < n || n == -1 {
		if n != -1 {
			opts.PerPage = n - len(list)
		}
		mrs, resp, err := lab.MergeRequests.ListProjectMergeRequests(p.ID, &opts)
		if err != nil {
			return nil, err
		}
		opts.Page = resp.NextPage
		list = append(list, mrs...)
		if resp.CurrentPage == resp.TotalPages {
			break
		}
	}
	return list, nil
}

// MRClose closes an mr on a GitLab project
func MRClose(pid interface{}, id int) error {
	mr, _, err := lab.MergeRequests.GetMergeRequest(pid, id, nil)
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

// MRListDiscussions retrieves the discussions (aka notes & comments) for a merge request
func MRListDiscussions(project string, mrNum int) ([]*gitlab.Discussion, error) {
	p, err := FindProject(project)
	if err != nil {
		return nil, err
	}

	discussions, _, err := lab.Discussions.ListMergeRequestDiscussions(p.ID, mrNum, &gitlab.ListMergeRequestDiscussionsOptions{})
	if err != nil {
		return nil, err
	}

	return discussions, nil
}

// MRRebase merges an mr on a GitLab project
func MRRebase(pid interface{}, id int) error {
	_, err := lab.MergeRequests.RebaseMergeRequest(pid, int(id))
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

// MRApprove approves an mr on a GitLab project
func MRApprove(pid interface{}, id int) error {
	_, _, err := lab.MergeRequestApprovals.ApproveMergeRequest(pid, id, &gitlab.ApproveMergeRequestOptions{})
	if err != nil {
		return err
	}
	return nil
}

// MRUnapprove Unapproves a previously approved mr on a GitLab project
func MRUnapprove(pid interface{}, id int) error {
	_, err := lab.MergeRequestApprovals.UnapproveMergeRequest(pid, id, nil)
	if err != nil {
		return err
	}
	return nil
}

// MRThumbUp places a thumb up/down on a merge request
func MRThumbUp(pid interface{}, id int) error {
	_, _, err := lab.AwardEmoji.CreateMergeRequestAwardEmoji(pid, id, &gitlab.CreateAwardEmojiOptions{
		Name: "thumbsup",
	})
	if err != nil {
		return err
	}
	return nil
}

// MRThumbDown places a thumb up/down on a merge request
func MRThumbDown(pid interface{}, id int) error {
	_, _, err := lab.AwardEmoji.CreateMergeRequestAwardEmoji(pid, id, &gitlab.CreateAwardEmojiOptions{
		Name: "thumbsdown",
	})
	if err != nil {
		return err
	}
	return nil
}

// IssueCreate opens a new issue on a GitLab project
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

// IssueUpdate edits an issue on a GitLab project
func IssueUpdate(project string, issueNum int, opts *gitlab.UpdateIssueOptions) (string, error) {
	p, err := FindProject(project)
	if err != nil {
		return "", err
	}

	issue, _, err := lab.Issues.UpdateIssue(p.ID, issueNum, opts)
	if err != nil {
		return "", err
	}
	return issue.WebURL, nil
}

// IssueCreateNote creates a new note on an issue and returns the note URL
func IssueCreateNote(project string, issueNum int, opts *gitlab.CreateIssueNoteOptions) (string, error) {
	p, err := FindProject(project)
	if err != nil {
		return "", err
	}

	note, _, err := lab.Notes.CreateIssueNote(p.ID, issueNum, opts)
	if err != nil {
		return "", err
	}

	// Unlike Issue, Note has no WebURL property, so we have to create it
	// ourselves from the project, noteable id and note id
	return fmt.Sprintf("%s/issues/%d#note_%d", p.WebURL, note.NoteableIID, note.ID), nil
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
func IssueList(project string, opts gitlab.ListProjectIssuesOptions, n int) ([]*gitlab.Issue, error) {
	if n == -1 {
		opts.PerPage = 100
	}
	p, err := FindProject(project)
	if err != nil {
		return nil, err
	}

	list, resp, err := lab.Issues.ListProjectIssues(p.ID, &opts)
	if err != nil {
		return nil, err
	}
	if resp.CurrentPage == resp.TotalPages {
		return list, nil
	}

	opts.Page = resp.NextPage
	for len(list) < n || n == -1 {
		if n != -1 {
			opts.PerPage = n - len(list)
		}
		issues, resp, err := lab.Issues.ListProjectIssues(p.ID, &opts)
		if err != nil {
			return nil, err
		}
		opts.Page = resp.NextPage
		list = append(list, issues...)
		if resp.CurrentPage == resp.TotalPages {
			break
		}
	}
	return list, nil
}

// IssueClose closes an issue on a GitLab project
func IssueClose(pid interface{}, id int) error {
	_, _, err := lab.Issues.UpdateIssue(pid, id, &gitlab.UpdateIssueOptions{
		StateEvent: gitlab.String("close"),
	})
	if err != nil {
		return err
	}
	return nil
}

// IssueListDiscussions retrieves the discussions (aka notes & comments) for an issue
func IssueListDiscussions(project string, issueNum int) ([]*gitlab.Discussion, error) {
	p, err := FindProject(project)
	if err != nil {
		return nil, err
	}

	discussions := []*gitlab.Discussion{}
	opt := &gitlab.ListIssueDiscussionsOptions{
		// 100 is the maximum allowed by the API
		PerPage: 100,
		Page:    1,
	}

	for {
		// get a page of discussions from the API ...
		d, resp, err := lab.Discussions.ListIssueDiscussions(p.ID, issueNum, opt)
		if err != nil {
			return nil, err
		}

		// ... and add them to our collection of discussions
		discussions = append(discussions, d...)

		// if we've seen all the pages, then we can break here
		if opt.Page >= resp.TotalPages {
			break
		}

		// otherwise, update the page number to get the next page.
		opt.Page = resp.NextPage
	}

	return discussions, nil
}

// GetCommit returns top Commit by ref (hash, branch or tag).
func GetCommit(pid interface{}, ref string) (*gitlab.Commit, error) {
	c, _, err := lab.Commits.GetCommit(pid, ref)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// LabelList gets a list of labels on a GitLab Project
func LabelList(project string) ([]*gitlab.Label, error) {
	p, err := FindProject(project)
	if err != nil {
		return nil, err
	}

	labels := []*gitlab.Label{}
	opt := &gitlab.ListLabelsOptions{
		ListOptions: gitlab.ListOptions{
			Page: 1,
		},
	}

	for {
		l, resp, err := lab.Labels.ListLabels(p.ID, opt)
		if err != nil {
			return nil, err
		}

		labels = append(labels, l...)

		// if we've seen all the pages, then we can break here
		if opt.Page >= resp.TotalPages {
			break
		}

		// otherwise, update the page number to get the next page.
		opt.Page = resp.NextPage
	}

	return labels, nil
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
func ProjectSnippetList(pid interface{}, opts gitlab.ListProjectSnippetsOptions, n int) ([]*gitlab.Snippet, error) {
	if n == -1 {
		opts.PerPage = 100
	}
	list, resp, err := lab.ProjectSnippets.ListSnippets(pid, &opts)
	if err != nil {
		return nil, err
	}
	if resp.CurrentPage == resp.TotalPages {
		return list, nil
	}
	opts.Page = resp.NextPage
	for len(list) < n || n == -1 {
		if n != -1 {
			opts.PerPage = n - len(list)
		}
		snips, resp, err := lab.ProjectSnippets.ListSnippets(pid, &opts)
		if err != nil {
			return nil, err
		}
		opts.Page = resp.NextPage
		list = append(list, snips...)
		if resp.CurrentPage == resp.TotalPages {
			break
		}
	}
	return list, nil
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
func SnippetList(opts gitlab.ListSnippetsOptions, n int) ([]*gitlab.Snippet, error) {
	if n == -1 {
		opts.PerPage = 100
	}
	list, resp, err := lab.Snippets.ListSnippets(&opts)
	if err != nil {
		return nil, err
	}
	if resp.CurrentPage == resp.TotalPages {
		return list, nil
	}
	opts.Page = resp.NextPage
	for len(list) < n || n == -1 {
		if n != -1 {
			opts.PerPage = n - len(list)
		}
		snips, resp, err := lab.Snippets.ListSnippets(&opts)
		if err != nil {
			return nil, err
		}
		opts.Page = resp.NextPage
		list = append(list, snips...)
		if resp.CurrentPage == resp.TotalPages {
			break
		}
	}
	return list, nil
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

// ProjectCreate creates a new project on GitLab
func ProjectCreate(opts *gitlab.CreateProjectOptions) (*gitlab.Project, error) {
	p, _, err := lab.Projects.CreateProject(opts)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// ProjectDelete creates a new project on GitLab
func ProjectDelete(pid interface{}) error {
	_, err := lab.Projects.DeleteProject(pid)
	if err != nil {
		return err
	}
	return nil
}

// ProjectList gets a list of projects on GitLab
func ProjectList(opts gitlab.ListProjectsOptions, n int) ([]*gitlab.Project, error) {
	list, resp, err := lab.Projects.ListProjects(&opts)
	if err != nil {
		return nil, err
	}
	if resp.CurrentPage == resp.TotalPages {
		return list, nil
	}
	opts.Page = resp.NextPage
	for len(list) < n || n == -1 {
		if n != -1 {
			opts.PerPage = n - len(list)
		}
		projects, resp, err := lab.Projects.ListProjects(&opts)
		if err != nil {
			return nil, err
		}
		opts.Page = resp.NextPage
		list = append(list, projects...)
		if resp.CurrentPage == resp.TotalPages {
			break
		}
	}
	return list, nil
}

// CIJobs returns a list of jobs in a pipeline for a given sha. The jobs are
// returned sorted by their CreatedAt time
func CIJobs(pid interface{}, sha string) ([]*gitlab.Job, error) {
	pipelines, _, err := lab.Pipelines.ListProjectPipelines(pid, &gitlab.ListProjectPipelinesOptions{
		SHA: gitlab.String(sha),
	})
	if len(pipelines) == 0 || err != nil {
		return nil, err
	}
	target := pipelines[0].ID
	opts := &gitlab.ListJobsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 500,
		},
	}
	list, resp, err := lab.Jobs.ListPipelineJobs(pid, target, opts)
	if err != nil {
		return nil, err
	}
	if resp.CurrentPage == resp.TotalPages {
		return list, nil
	}
	opts.Page = resp.NextPage
	for {
		jobs, resp, err := lab.Jobs.ListPipelineJobs(pid, target, opts)
		if err != nil {
			return nil, err
		}
		opts.Page = resp.NextPage
		list = append(list, jobs...)
		if resp.CurrentPage == resp.TotalPages {
			break
		}
	}
	return list, nil
}

// CITrace searches by name for a job and returns its trace file. The trace is
// static so may only be a portion of the logs if the job is till running. If
// no name is provided job is picked using the first available:
// 1. Last Running Job
// 2. First Pending Job
// 3. Last Job in Pipeline
func CITrace(pid interface{}, sha, name string) (io.Reader, *gitlab.Job, error) {
	jobs, err := CIJobs(pid, sha)
	if len(jobs) == 0 || err != nil {
		return nil, nil, err
	}
	var (
		job          *gitlab.Job
		lastRunning  *gitlab.Job
		firstPending *gitlab.Job
	)

	for _, j := range jobs {
		if j.Status == "running" {
			lastRunning = j
		}
		if j.Status == "pending" && firstPending == nil {
			firstPending = j
		}
		if j.Name == name {
			job = j
			// don't break because there may be a newer version of the job
		}
	}
	if job == nil {
		job = lastRunning
	}
	if job == nil {
		job = firstPending
	}
	if job == nil {
		job = jobs[len(jobs)-1]
	}
	r, _, err := lab.Jobs.GetTraceFile(pid, job.ID)
	if err != nil {
		return nil, job, err
	}

	return r, job, err
}

// CIPlayOrRetry runs a job either by playing it for the first time or by
// retrying it based on the currently known job state
func CIPlayOrRetry(pid interface{}, jobID int, status string) (*gitlab.Job, error) {
	switch status {
	case "pending", "running":
		return nil, nil
	case "manual":
		j, _, err := lab.Jobs.PlayJob(pid, jobID)
		if err != nil {
			return nil, err
		}
		return j, nil
	default:

		j, _, err := lab.Jobs.RetryJob(pid, jobID)
		if err != nil {
			return nil, err
		}

		return j, nil
	}
}

// CICancel cancels a job for a given project by its ID.
func CICancel(pid interface{}, jobID int) (*gitlab.Job, error) {
	j, _, err := lab.Jobs.CancelJob(pid, jobID)
	if err != nil {
		return nil, err
	}
	return j, nil
}

// CICreate creates a pipeline for given ref
func CICreate(pid interface{}, opts *gitlab.CreatePipelineOptions) (*gitlab.Pipeline, error) {
	p, _, err := lab.Pipelines.CreatePipeline(pid, opts)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// CITrigger triggers a pipeline for given ref
func CITrigger(pid interface{}, opts gitlab.RunPipelineTriggerOptions) (*gitlab.Pipeline, error) {
	p, _, err := lab.PipelineTriggers.RunPipelineTrigger(pid, &opts)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// UserIDFromUsername returns the associated Users ID in GitLab. This is useful
// for API calls that allow you to reference a user, but only by ID.
func UserIDFromUsername(username string) (int, error) {
	us, _, err := lab.Users.ListUsers(&gitlab.ListUsersOptions{
		Username: gitlab.String(username),
	})
	if err != nil || len(us) == 0 {
		return -1, err
	}
	return us[0].ID, nil
}

// Labels converts a []string into a non-nil *gitlab.Labels.
func Labels(labels []string) *gitlab.Labels {
	l := gitlab.Labels(labels)
	return &l
}

// AddMRDiscussionNote adds a note to an existing MR discussion on GitLab
func AddMRDiscussionNote(project string, mrNum int, discussionID string, body string) (string, error) {
	p, err := FindProject(project)
	if err != nil {
		return "", err
	}

	opts := &gitlab.AddMergeRequestDiscussionNoteOptions{
		Body: &body,
	}

	note, _, err := lab.Discussions.AddMergeRequestDiscussionNote(p.ID, mrNum, discussionID, opts)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/merge_requests/%d#note_%d", p.WebURL, note.NoteableIID, note.ID), nil
}

// AddIssueDiscussionNote adds a note to an existing issue discussion on GitLab
func AddIssueDiscussionNote(project string, issueNum int, discussionID string, body string) (string, error) {
	p, err := FindProject(project)
	if err != nil {
		return "", err
	}

	opts := &gitlab.AddIssueDiscussionNoteOptions{
		Body: &body,
	}

	note, _, err := lab.Discussions.AddIssueDiscussionNote(p.ID, issueNum, discussionID, opts)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/issues/%d#note_%d", p.WebURL, note.NoteableIID, note.ID), nil
}

func UpdateIssueDiscussionNote(project string, issueNum int, discussionID string, noteID int, body string) (string, error) {
	p, err := FindProject(project)
	if err != nil {
		return "", err
	}
	opts := &gitlab.UpdateIssueDiscussionNoteOptions{
		Body: &body,
	}

	note, _, err := lab.Discussions.UpdateIssueDiscussionNote(p.ID, issueNum, discussionID, noteID, opts)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/issues/%d#note_%d", p.WebURL, note.NoteableIID, note.ID), nil
}

func UpdateMRDiscussionNote(project string, issueNum int, discussionID string, noteID int, body string) (string, error) {
	p, err := FindProject(project)
	if err != nil {
		return "", err
	}
	opts := &gitlab.UpdateMergeRequestDiscussionNoteOptions{
		Body: &body,
	}

	note, _, err := lab.Discussions.UpdateMergeRequestDiscussionNote(p.ID, issueNum, discussionID, noteID, opts)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/merge_requests/%d#note_%d", p.WebURL, note.NoteableIID, note.ID), nil
}

func ListMRsClosingIssue(project string, issueNum int) ([]int, error) {

	var retArray []int

	p, err := FindProject(project)
	if err != nil {
		return retArray, err
	}

	mrs, _, err := lab.Issues.ListMergeRequestsClosingIssue(p.ID, issueNum, nil, nil)
	if err != nil {
		return retArray, err
	}

	for _, mr := range mrs {
		retArray = append(retArray, mr.IID)
	}

	return retArray, nil
}

func ListMRsRelatedToIssue(project string, issueNum int) ([]int, error) {

	var retArray []int

	p, err := FindProject(project)
	if err != nil {
		return retArray, err
	}

	mrs, _, err := lab.Issues.ListMergeRequestsRelatedToIssue(p.ID, issueNum, nil, nil)
	if err != nil {
		return retArray, err
	}

	for _, mr := range mrs {
		retArray = append(retArray, mr.IID)
	}

	return retArray, nil
}

func ListIssuesClosedOnMerge(project string, mrNum int) ([]int, error) {
	var retArray []int

	p, err := FindProject(project)
	if err != nil {
		return retArray, err
	}

	issues, _, err := lab.MergeRequests.GetIssuesClosedOnMerge(p.ID, mrNum, nil, nil)
	if err != nil {
		return retArray, err
	}

	for _, issue := range issues {
		retArray = append(retArray, issue.IID)
	}

	return retArray, nil

}
