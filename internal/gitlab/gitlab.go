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
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/config"
	"github.com/zaquestion/lab/internal/git"
	"github.com/zaquestion/lab/internal/logger"
)

// Get internal lab logger instance
var log = logger.GetInstance()

// 100 is the maximum allowed by the API
const maxItemsPerPage = 100

var (
	// ErrActionRepeated is returned when a GitLab action is executed again.  For example
	// this can be returned when an MR is approved twice.
	ErrActionRepeated = errors.New("GitLab action repeated")
	// ErrGroupNotFound is returned when a GitLab group cannot be found.
	ErrGroupNotFound = errors.New("GitLab group not found")
	// ErrNotModified is returned when adding an already existing item to a Todo list
	ErrNotModified = errors.New("Not Modified")
	// ErrProjectNotFound is returned when a GitLab project cannot be found.
	ErrProjectNotFound = errors.New("GitLab project not found, verify you have access to the requested resource")
	// ErrStatusForbidden is returned when attempting to access a GitLab project with insufficient permissions
	ErrStatusForbidden = errors.New("Insufficient permissions for GitLab project")
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

// UserID get the current user ID from gitlab server
func UserID() (int, error) {
	u, _, err := lab.Users.CurrentUser()
	if err != nil {
		return 0, err
	}
	return u.ID, nil
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

	lab, _ = gitlab.NewClient(token, gitlab.WithHTTPClient(httpClient), gitlab.WithBaseURL(host+"/api/v4"), gitlab.WithCustomLeveledLogger(log))
}

// InitWithCustomCA open the HTTP client using a custom CA file (a self signed
// one for instance) instead of relying only on those installed in the current
// system database
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

var localProjects map[string]*gitlab.Project = make(map[string]*gitlab.Project)

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

// Fork creates a user fork of a GitLab project using the specified protocol
func Fork(project string, opts *gitlab.ForkProjectOptions, useHTTP bool, wait bool) (string, error) {
	if !strings.Contains(project, "/") {
		return "", errors.New("remote must include namespace")
	}
	parts := strings.Split(project, "/")

	// See if a fork already exists in the destination
	name := parts[len(parts)-1]
	namespace := ""
	if opts != nil {
		var (
			optName      = *(opts.Name)
			optNamespace = *(opts.Namespace)
			optPath      = *(opts.Path)
		)

		if optNamespace != "" {
			namespace = optNamespace + "/"
		}
		// Project name takes precedence over path for finding a project
		// on Gitlab through API
		if optName != "" {
			name = optName
		} else if optPath != "" {
			name = optPath
		} else {
			opts.Name = gitlab.String(name)
		}
	}

	target, err := FindProject(namespace + name)
	if err == nil {
		// Check if it isn't the same project being requested
		if target.PathWithNamespace == project {
			errMsg := "not possible to fork a project from the same namespace and name"
			return "", errors.New(errMsg)
		}

		// Check if it isn't a non-fork project, meaning the user has
		// access to a project with same namespace/name
		if target.ForkedFromProject == nil {
			errMsg := fmt.Sprintf("\"%s\" project already taken\n", target.PathWithNamespace)
			return "", errors.New(errMsg)
		}

		// Check if it isn't already a fork for another project
		if target.ForkedFromProject != nil &&
			target.ForkedFromProject.PathWithNamespace != project {
			errMsg := fmt.Sprintf("\"%s\" fork already taken for a different project",
				target.PathWithNamespace)
			return "", errors.New(errMsg)
		}

		// Project already forked and found
		urlToRepo := target.SSHURLToRepo
		if useHTTP {
			urlToRepo = target.HTTPURLToRepo
		}
		return urlToRepo, nil
	} else if err != nil && err != ErrProjectNotFound {
		return "", err
	}

	target, err = FindProject(project)
	if err != nil {
		return "", err
	}

	// Now that we have the "wait" opt, don't let the user in the hope that
	// something is running.
	fmt.Printf("Forking %s project...\n", project)
	fork, _, err := lab.Projects.ForkProject(target.ID, opts)
	if err != nil {
		return "", err
	}

	// Busy-wait approach for checking the import_status of the fork.
	// References:
	//   https://docs.gitlab.com/ce/api/projects.html#fork-project
	//   https://docs.gitlab.com/ee/api/project_import_export.html#import-status
	status, _, err := lab.ProjectImportExport.ImportStatus(fork.ID, nil)
	if err != nil {
		log.Infof("Impossible to get fork status: %s\n", err)
	} else {
		if wait {
			for {
				if status.ImportStatus == "finished" {
					break
				}
				status, _, err = lab.ProjectImportExport.ImportStatus(fork.ID, nil)
				if err != nil {
					log.Fatal(err)
				}
				time.Sleep(2 * time.Second)
			}
		} else if status.ImportStatus != "finished" {
			err = errors.New("not finished")
		}
	}

	urlToRepo := fork.SSHURLToRepo
	if useHTTP {
		urlToRepo = fork.HTTPURLToRepo
	}
	return urlToRepo, err
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

// MRDelete deletes an merge request on a GitLab project
func MRDelete(project string, mrNum int) error {
	p, err := FindProject(project)
	if err != nil {
		return err
	}
	resp, err := lab.MergeRequests.DeleteMergeRequest(p.ID, mrNum)
	if resp != nil && resp.StatusCode == http.StatusForbidden {
		return ErrStatusForbidden
	}
	if err != nil {
		return err
	}

	return nil
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
func MRGet(project interface{}, mrNum int) (*gitlab.MergeRequest, error) {
	mr, _, err := lab.MergeRequests.GetMergeRequest(project, mrNum, nil)
	if err != nil {
		return nil, err
	}

	return mr, nil
}

// MRList lists the MRs on a GitLab project
func MRList(project string, opts gitlab.ListProjectMergeRequestsOptions, n int) ([]*gitlab.MergeRequest, error) {
	if n == -1 {
		opts.PerPage = maxItemsPerPage
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

// MRReopen reopen an already close mr on a GitLab project
func MRReopen(pid interface{}, id int) error {
	mr, _, err := lab.MergeRequests.GetMergeRequest(pid, id, nil)
	if err != nil {
		return err
	}
	if mr.State == "opened" {
		return fmt.Errorf("mr not closed")
	}
	_, _, err = lab.MergeRequests.UpdateMergeRequest(pid, int(id), &gitlab.UpdateMergeRequestOptions{
		StateEvent: gitlab.String("reopen"),
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

	discussions := []*gitlab.Discussion{}
	opt := &gitlab.ListMergeRequestDiscussionsOptions{
		// 100 is the maximum allowed by the API
		PerPage: maxItemsPerPage,
	}

	for {
		// get a page of discussions from the API ...
		d, resp, err := lab.Discussions.ListMergeRequestDiscussions(p.ID, mrNum, opt)
		if err != nil {
			return nil, err
		}

		// ... and add them to our collection of discussions
		discussions = append(discussions, d...)

		// if we've seen all the pages, then we can break here
		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		// otherwise, update the page number to get the next page.
		opt.Page = resp.NextPage
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
func MRMerge(pid interface{}, id int, opts *gitlab.AcceptMergeRequestOptions) error {
	_, _, err := lab.MergeRequests.AcceptMergeRequest(pid, int(id), opts)
	if err != nil {
		return err
	}
	return nil
}

// MRApprove approves an mr on a GitLab project
func MRApprove(pid interface{}, id int) error {
	_, resp, err := lab.MergeRequestApprovals.ApproveMergeRequest(pid, id, &gitlab.ApproveMergeRequestOptions{})
	if resp != nil && resp.StatusCode == http.StatusForbidden {
		return ErrStatusForbidden
	}
	if resp != nil && resp.StatusCode == http.StatusUnauthorized {
		// returns 401 if the MR has already been approved
		return ErrActionRepeated
	}
	if err != nil {
		return err
	}
	return nil
}

// MRUnapprove Unapproves a previously approved mr on a GitLab project
func MRUnapprove(pid interface{}, id int) error {
	resp, err := lab.MergeRequestApprovals.UnapproveMergeRequest(pid, id, nil)
	if resp != nil && resp.StatusCode == http.StatusForbidden {
		return ErrStatusForbidden
	}
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		// returns 404 if the MR has already been unapproved
		return ErrActionRepeated
	}
	if err != nil {
		return err
	}
	return nil
}

// MRSubscribe subscribes to an mr on a GitLab project
func MRSubscribe(pid interface{}, id int) error {
	_, resp, err := lab.MergeRequests.SubscribeToMergeRequest(pid, id, nil)
	if resp != nil && resp.StatusCode == http.StatusNotModified {
		return errors.New("Already subscribed")
	}
	if err != nil {
		return err
	}
	return nil
}

// MRUnsubscribe unsubscribes from a previously mr on a GitLab project
func MRUnsubscribe(pid interface{}, id int) error {
	_, resp, err := lab.MergeRequests.UnsubscribeFromMergeRequest(pid, id, nil)
	if resp != nil && resp.StatusCode == http.StatusNotModified {
		return errors.New("Not subscribed")
	}
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
func IssueGet(project interface{}, issueNum int) (*gitlab.Issue, error) {
	issue, _, err := lab.Issues.GetIssue(project, issueNum)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

// IssueList gets a list of issues on a GitLab Project
func IssueList(project string, opts gitlab.ListProjectIssuesOptions, n int) ([]*gitlab.Issue, error) {
	if n == -1 {
		opts.PerPage = maxItemsPerPage
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
	issue, _, err := lab.Issues.GetIssue(pid, id)
	if err != nil {
		return err
	}
	if issue.State == "closed" {
		return fmt.Errorf("issue already closed")
	}
	_, _, err = lab.Issues.UpdateIssue(pid, id, &gitlab.UpdateIssueOptions{
		StateEvent: gitlab.String("close"),
	})
	if err != nil {
		return err
	}
	return nil
}

// IssueDuplicate closes an issue as duplicate of another
func IssueDuplicate(pid interface{}, id int, dupID string) error {
	// Not exposed in API, go through quick action
	body := "/duplicate " + dupID

	_, _, err := lab.Notes.CreateIssueNote(pid, id, &gitlab.CreateIssueNoteOptions{
		Body: &body,
	})
	if err != nil {
		return errors.Errorf("Failed to close issue #%d as duplicate of %s", id, dupID)
	}

	issue, _, err := lab.Issues.GetIssue(pid, id)
	if issue == nil || issue.State != "closed" {
		return errors.Errorf("Failed to close issue #%d as duplicate of %s", id, dupID)
	}
	return nil
}

// IssueReopen reopens a closed issue
func IssueReopen(pid interface{}, id int) error {
	issue, _, err := lab.Issues.GetIssue(pid, id)
	if err != nil {
		return err
	}
	if issue.State == "opened" {
		return fmt.Errorf("issue not closed")
	}
	_, _, err = lab.Issues.UpdateIssue(pid, id, &gitlab.UpdateIssueOptions{
		StateEvent: gitlab.String("reopen"),
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
		PerPage: maxItemsPerPage,
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
		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		// otherwise, update the page number to get the next page.
		opt.Page = resp.NextPage
	}

	return discussions, nil
}

// IssueSubscribe subscribes to an issue on a GitLab project
func IssueSubscribe(pid interface{}, id int) error {
	_, resp, err := lab.Issues.SubscribeToIssue(pid, id, nil)
	if resp != nil && resp.StatusCode == http.StatusNotModified {
		return errors.New("Already subscribed")
	}
	if err != nil {
		return err
	}
	return nil
}

// IssueUnsubscribe unsubscribes from an issue on a GitLab project
func IssueUnsubscribe(pid interface{}, id int) error {
	_, resp, err := lab.Issues.UnsubscribeFromIssue(pid, id, nil)
	if resp != nil && resp.StatusCode == http.StatusNotModified {
		return errors.New("Not subscribed")
	}
	if err != nil {
		return err
	}
	return nil
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
			PerPage: maxItemsPerPage,
		},
	}

	for {
		l, resp, err := lab.Labels.ListLabels(p.ID, opt)
		if err != nil {
			return nil, err
		}

		labels = append(labels, l...)

		// if we've seen all the pages, then we can break here
		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		// otherwise, update the page number to get the next page.
		opt.Page = resp.NextPage
	}

	return labels, nil
}

// LabelCreate creates a new project label
func LabelCreate(project string, opts *gitlab.CreateLabelOptions) error {
	p, err := FindProject(project)
	if err != nil {
		return err
	}

	_, _, err = lab.Labels.CreateLabel(p.ID, opts)
	return err
}

// LabelDelete removes a project label
func LabelDelete(project, name string) error {
	p, err := FindProject(project)
	if err != nil {
		return err
	}

	_, err = lab.Labels.DeleteLabel(p.ID, &gitlab.DeleteLabelOptions{
		Name: &name,
	})
	return err
}

// BranchList get all branches from the project that somehow matches the
// requested options
func BranchList(project string, opts *gitlab.ListBranchesOptions) ([]*gitlab.Branch, error) {
	p, err := FindProject(project)
	if err != nil {
		return nil, err
	}

	branches := []*gitlab.Branch{}
	for {
		bList, resp, err := lab.Branches.ListBranches(p.ID, opts)
		if err != nil {
			return nil, err
		}
		branches = append(branches, bList...)

		if resp.CurrentPage >= resp.TotalPages {
			break
		}
		opts.Page = resp.NextPage
	}

	return branches, nil
}

// MilestoneGet get a specific milestone from the list of available ones
func MilestoneGet(project string, name string) (*gitlab.Milestone, error) {
	opts := &gitlab.ListMilestonesOptions{
		Search: &name,
	}
	milestones, _ := MilestoneList(project, opts)

	switch len(milestones) {
	case 1:
		return milestones[0], nil
	case 0:
		return nil, errors.Errorf("Milestone '%s' not found", name)
	default:
		return nil, errors.Errorf("Milestone '%s' is ambiguous", name)
	}
}

// MilestoneList gets a list of milestones on a GitLab Project
func MilestoneList(project string, opt *gitlab.ListMilestonesOptions) ([]*gitlab.Milestone, error) {
	p, err := FindProject(project)
	if err != nil {
		return nil, err
	}

	milestones := []*gitlab.Milestone{}
	for {
		m, resp, err := lab.Milestones.ListMilestones(p.ID, opt)
		if err != nil {
			return nil, err
		}

		milestones = append(milestones, m...)

		// if we've seen all the pages, then we can break here
		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		// otherwise, update the page number to get the next page.
		opt.Page = resp.NextPage
	}

	if p.Namespace.Kind != "group" {
		return milestones, nil
	}

	// get inherited milestones from group; in the future, we'll be able to use the
	// IncludeParentMilestones option with ListMilestones()
	includeParents := true
	gopt := &gitlab.ListGroupMilestonesOptions{
		IIDs:                    opt.IIDs,
		Title:                   opt.Title,
		State:                   opt.State,
		Search:                  opt.Search,
		IncludeParentMilestones: &includeParents,
	}

	for {
		groupMilestones, resp, err := lab.GroupMilestones.ListGroupMilestones(p.Namespace.ID, gopt)
		if err != nil {
			return nil, err
		}

		for _, m := range groupMilestones {
			milestones = append(milestones, &gitlab.Milestone{
				ID:          m.ID,
				IID:         m.IID,
				Title:       m.Title,
				Description: m.Description,
				StartDate:   m.StartDate,
				DueDate:     m.DueDate,
				State:       m.State,
				UpdatedAt:   m.UpdatedAt,
				CreatedAt:   m.CreatedAt,
				Expired:     m.Expired,
			})
		}

		// if we've seen all the pages, then we can break here
		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		// otherwise, update the page number to get the next page.
		gopt.Page = resp.NextPage
	}

	return milestones, nil
}

// MilestoneCreate creates a new project milestone
func MilestoneCreate(project string, opts *gitlab.CreateMilestoneOptions) error {
	p, err := FindProject(project)
	if err != nil {
		return err
	}

	_, _, err = lab.Milestones.CreateMilestone(p.ID, opts)
	return err
}

// MilestoneDelete deletes a project milestone
func MilestoneDelete(project, name string) error {
	milestone, err := MilestoneGet(project, name)
	if err != nil {
		return err
	}

	_, err = lab.Milestones.DeleteMilestone(milestone.ProjectID, milestone.ID)
	return err
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
		opts.PerPage = maxItemsPerPage
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
		opts.PerPage = maxItemsPerPage
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

// JobStruct maps the project ID to which a certain job belongs to.
// It's needed due to multi-projects pipeline, which allows jobs from
// different projects be triggered by the current project.
// CIJob() is currently the function handling the mapping.
type JobStruct struct {
	Job *gitlab.Job
	// A project ID can either be a string or an integer
	ProjectID interface{}
}
type jobSorter struct{ Jobs []JobStruct }

func (s jobSorter) Len() int      { return len(s.Jobs) }
func (s jobSorter) Swap(i, j int) { s.Jobs[i], s.Jobs[j] = s.Jobs[j], s.Jobs[i] }
func (s jobSorter) Less(i, j int) bool {
	return time.Time(*s.Jobs[i].Job.CreatedAt).Before(time.Time(*s.Jobs[j].Job.CreatedAt))
}

// GroupSearch searches for a namespace on GitLab
func GroupSearch(query string) (*gitlab.Group, error) {
	if query == "" {
		return nil, errors.New("query is empty")
	}
	groups := strings.Split(query, "/")
	list, _, err := lab.Groups.SearchGroup(groups[0])
	if err != nil {
		return nil, err
	}
	// SearchGroup doesn't return error if group isn't found. We need to do
	// it ourselves.
	if len(list) == 0 {
		return nil, ErrGroupNotFound
	}
	// if we found a group and we aren't looking for a subgroup
	if len(list) > 0 && len(groups) == 1 {
		return list[0], nil
	}
	list, _, err = lab.Groups.ListDescendantGroups(list[0].ID, &gitlab.ListDescendantGroupsOptions{
		Search: gitlab.String(groups[len(groups)-1]),
	})
	if err != nil {
		return nil, err
	}

	for _, g := range list {
		fmt.Println(g.FullPath)
		if g.FullPath == query {
			return g, nil
		}
	}

	return nil, errors.Errorf("Group '%s' not found", query)
}

// CIJobs returns a list of jobs in the pipeline with given id.
// This function by default doesn't follow bridge jobs.
// The jobs are returned sorted by their CreatedAt time
func CIJobs(pid interface{}, id int, followBridge bool) ([]JobStruct, error) {
	opts := &gitlab.ListJobsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: maxItemsPerPage,
		},
	}

	// First we get the jobs with direct relation to the actual project
	list := make([]JobStruct, 0)
	for {
		jobs, resp, err := lab.Jobs.ListPipelineJobs(pid, id, opts)
		if err != nil {
			return nil, err
		}

		for _, job := range jobs {
			list = append(list, JobStruct{job, pid})
		}

		opts.Page = resp.NextPage
		if resp.CurrentPage == resp.TotalPages {
			break
		}
	}

	// It's also possible the pipelines are bridges to other project's
	// pipelines (multi-project pipeline).
	// Reference:
	//     https://docs.gitlab.com/ee/ci/multi_project_pipelines.html
	if followBridge {
		// A project can have multiple bridge jobs
		bridgeList := make([]*gitlab.Bridge, 0)
		for {
			bridges, resp, err := lab.Jobs.ListPipelineBridges(pid, id, opts)
			if err != nil {
				return nil, err
			}

			opts.Page = resp.NextPage
			bridgeList = append(bridgeList, bridges...)
			if resp.CurrentPage == resp.TotalPages {
				break
			}
		}

		for _, bridge := range bridgeList {
			// Unfortunately the GitLab API doesn't exposes the project ID nor name that the
			// bridge job points to, since it might be extarnal to the config core.host
			// hostname, hence the WebURL is exposed.
			// With that, and considering we don't want to support anything outside the
			// core.host, we need to massage the WebURL to get the project name that we can
			// search for.
			// WebURL format:
			//   <core.host>/<bridged-project-name-with-namespace>/-/pipelines/<id>
			host := config.MainConfig.GetString("core.host")
			projectName := strings.Replace(bridge.DownstreamPipeline.WebURL, host+"/", "", 1)
			pipelineText := fmt.Sprintf("/-/pipelines/%d", bridge.DownstreamPipeline.ID)
			projectName = strings.Replace(projectName, pipelineText, "", 1)

			p, err := FindProject(projectName)
			if err != nil {
				continue
			}

			// Switch to the new project name and downstream pipeline id
			pid = p.PathWithNamespace
			id = bridge.DownstreamPipeline.ID

			for {
				// Get the list of bridged jobs and append to the original list
				jobs, resp, err := lab.Jobs.ListPipelineJobs(pid, id, opts)
				if err != nil {
					return nil, err
				}

				for _, job := range jobs {
					list = append(list, JobStruct{job, pid})
				}

				opts.Page = resp.NextPage
				if resp.CurrentPage == resp.TotalPages {
					break
				}
			}
		}
	}

	// ListPipelineJobs returns jobs sorted by ID in descending order,
	// while we want them to be ordered chronologically
	sort.Sort(jobSorter{list})

	return list, nil
}

// CITrace searches by name for a job and returns its trace file. The trace is
// static so may only be a portion of the logs if the job is till running. If
// no name is provided job is picked using the first available:
// 1. Last Running Job
// 2. First Pending Job
// 3. Last Job in Pipeline
func CITrace(pid interface{}, id int, name string, followBridge bool) (io.Reader, *gitlab.Job, error) {
	jobs, err := CIJobs(pid, id, followBridge)
	if len(jobs) == 0 || err != nil {
		return nil, nil, err
	}
	var (
		job          *gitlab.Job
		lastRunning  *gitlab.Job
		firstPending *gitlab.Job
	)

	for _, jobStruct := range jobs {
		// Switch to the project ID that owns the job (for a bridge case)
		pid = jobStruct.ProjectID
		j := jobStruct.Job
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
		job = jobs[len(jobs)-1].Job
	}

	r, _, err := lab.Jobs.GetTraceFile(pid, job.ID)
	if err != nil {
		return nil, job, err
	}

	return r, job, err
}

// CIArtifacts searches by name for a job and returns its artifacts archive
// together with the upstream filename. If path is specified and refers to
// a single file within the artifacts archive, that file is returned instead.
// If no name is provided, the last job with an artifacts file is picked.
func CIArtifacts(pid interface{}, id int, name, path string, followBridge bool) (io.Reader, string, error) {
	jobs, err := CIJobs(pid, id, followBridge)
	if len(jobs) == 0 || err != nil {
		return nil, "", err
	}
	var (
		job               *gitlab.Job
		lastWithArtifacts *gitlab.Job
	)

	for _, jobStruct := range jobs {
		// Switch to the project ID that owns the job (for a bridge case)
		pid = jobStruct.ProjectID
		j := jobStruct.Job
		if j.ArtifactsFile.Filename != "" {
			lastWithArtifacts = j
		}
		if j.Name == name {
			job = j
			// don't break because there may be a newer version of the job
		}
	}
	if job == nil {
		job = lastWithArtifacts
	}
	if job == nil {
		return nil, "", fmt.Errorf("Could not find any jobs with artifacts")
	}

	var (
		r       io.Reader
		outpath string
	)

	if job.ArtifactsFile.Filename == "" {
		return nil, "", fmt.Errorf("Job %d has no artifacts", job.ID)
	}

	fmt.Println("Downloading artifacts...")
	if path != "" {
		r, _, err = lab.Jobs.DownloadSingleArtifactsFile(pid, job.ID, path, nil)
		outpath = filepath.Base(path)
	} else {
		r, _, err = lab.Jobs.GetJobArtifacts(pid, job.ID, nil)
		outpath = job.ArtifactsFile.Filename
	}

	if err != nil {
		return nil, "", err
	}

	return r, outpath, nil
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

// UserIDFromEmail returns the associated Users ID in GitLab. This is useful
// for API calls that allow you to reference a user, but only by ID.
func UserIDFromEmail(email string) (int, error) {
	us, _, err := lab.Users.ListUsers(&gitlab.ListUsersOptions{
		Search: gitlab.String(email),
	})
	if err != nil || len(us) == 0 {
		return -1, err
	}
	return us[0].ID, nil
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

// UpdateIssueDiscussionNote updates a specific discussion or note in the
// specified issue number
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

// UpdateMRDiscussionNote updates a specific discussion or note in the
// specified MR ID.
func UpdateMRDiscussionNote(project string, mrNum int, discussionID string, noteID int, body string) (string, error) {
	p, err := FindProject(project)
	if err != nil {
		return "", err
	}
	opts := &gitlab.UpdateMergeRequestDiscussionNoteOptions{
		Body: &body,
	}

	note, _, err := lab.Discussions.UpdateMergeRequestDiscussionNote(p.ID, mrNum, discussionID, noteID, opts)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/merge_requests/%d#note_%d", p.WebURL, note.NoteableIID, note.ID), nil
}

// ListMRsClosingIssue returns a list of MR IDs that has relation to an issue
// being closed
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

// ListMRsRelatedToIssue return a list of MR IDs that has any relations to a
// certain issue
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

// ListIssuesClosedOnMerge retuns a list of issue numbers that were closed by
// an MR being merged
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

// MoveIssue moves one issue from one project to another
func MoveIssue(project string, issueNum int, dest string) (string, error) {
	srcProject, err := FindProject(project)
	if err != nil {
		return "", err
	}

	destProject, err := FindProject(dest)
	if err != nil {
		return "", err
	}

	opts := &gitlab.MoveIssueOptions{
		ToProjectID: &destProject.ID,
	}

	issue, _, err := lab.Issues.MoveIssue(srcProject.ID, issueNum, opts)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/issues/%d", destProject.WebURL, issue.IID), nil
}

// GetMRApprovalsConfiguration returns the current MR approval rule
func GetMRApprovalsConfiguration(project string, mrNum int) (*gitlab.MergeRequestApprovals, error) {
	p, err := FindProject(project)
	if err != nil {
		return nil, err
	}

	configuration, _, err := lab.MergeRequestApprovals.GetConfiguration(p.ID, mrNum)
	if err != nil {
		return nil, err
	}

	return configuration, err
}

// ResolveMRDiscussion resolves a discussion (blocking thread) based on its ID
func ResolveMRDiscussion(project string, mrNum int, discussionID string, noteID int) (string, error) {
	p, err := FindProject(project)
	if err != nil {
		return "", err
	}

	opts := &gitlab.ResolveMergeRequestDiscussionOptions{
		Resolved: gitlab.Bool(true),
	}

	discussion, _, err := lab.Discussions.ResolveMergeRequestDiscussion(p.ID, mrNum, discussionID, opts)
	if err != nil {
		return discussion.ID, err
	}
	return fmt.Sprintf("Resolved %s/merge_requests/%d#note_%d", p.WebURL, mrNum, noteID), nil
}

// TodoList retuns a list of *gitlab.Todo refering to user's Todo list
func TodoList(opts gitlab.ListTodosOptions, n int) ([]*gitlab.Todo, error) {
	if n == -1 {
		opts.PerPage = maxItemsPerPage
	}

	list, resp, err := lab.Todos.ListTodos(&opts)
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

		todos, resp, err := lab.Todos.ListTodos(&opts)
		if err != nil {
			return nil, err
		}

		opts.Page = resp.NextPage
		list = append(list, todos...)
		if resp.CurrentPage == resp.TotalPages {
			break
		}
	}

	return list, nil
}

// TodoMarkDone marks a specific Todo as done
func TodoMarkDone(todoNum int) error {
	_, err := lab.Todos.MarkTodoAsDone(todoNum)
	if err != nil {
		return err
	}
	return nil
}

// TodoMarkAllDone marks all Todos items as done
func TodoMarkAllDone() error {
	_, err := lab.Todos.MarkAllTodosAsDone()
	if err != nil {
		return err
	}
	return nil
}

// TodoMRCreate create a Todo item for an specific MR
func TodoMRCreate(project string, mrNum int) (int, error) {
	p, err := FindProject(project)
	if err != nil {
		return 0, err
	}

	todo, resp, err := lab.MergeRequests.CreateTodo(p.ID, mrNum)
	if err != nil {
		if resp.StatusCode == http.StatusNotModified {
			return 0, ErrNotModified
		}
		return 0, err
	}
	return todo.ID, nil
}

// TodoIssueCreate create a Todo item for an specific Issue
func TodoIssueCreate(project string, issueNum int) (int, error) {
	p, err := FindProject(project)
	if err != nil {
		return 0, err
	}

	todo, resp, err := lab.Issues.CreateTodo(p.ID, issueNum)
	if err != nil {
		if resp.StatusCode == http.StatusNotModified {
			return 0, ErrNotModified
		}
		return 0, err
	}
	return todo.ID, nil
}
