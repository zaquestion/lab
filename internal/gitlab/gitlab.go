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
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	gitlab "github.com/xanzy/go-gitlab"
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
	if len(_host) > 0 && _host[len(_host)-1] == '/' {
		_host = _host[:len(_host)-1]
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
	if len(_host) > 0 && _host[len(_host)-1] == '/' {
		_host = _host[:len(_host)-1]
	}
	host = _host
	user = _user
	token = _token

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

func parseID(id interface{}) (string, error) {
	var strID string

	switch v := id.(type) {
	case int:
		strID = strconv.Itoa(v)
	case string:
		strID = v
	default:
		return "", fmt.Errorf("unknown id type %#v", id)
	}

	return strID, nil
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
	content, err := ioutil.ReadFile(tmplFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ""
		}
		log.Fatal(err)
	}

	return strings.TrimSpace(string(content))
}

var localProjects map[string]*gitlab.Project = make(map[string]*gitlab.Project)

// GetProject looks up a Gitlab project by ID.
func GetProject(projID interface{}) (*gitlab.Project, error) {
	target, resp, err := lab.Projects.GetProject(projID, nil)
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
func FindProject(projID interface{}) (*gitlab.Project, error) {
	var (
		id     string
		search string
	)

	switch v := projID.(type) {
	case int:
		// If the project number is used directly, don't "guess" anything
		id = strconv.Itoa(v)
		search = id
	case string:
		id = v
		search = id
		// If the project name is used, check if it already has the
		// namespace (already have a slash '/' in the name) or try to guess
		// it's on user's own namespace.
		if !strings.Contains(id, "/") {
			search = user + "/" + id
		}
	}

	if target, ok := localProjects[id]; ok {
		return target, nil
	}

	target, err := GetProject(search)
	if err != nil {
		return nil, err
	}

	// fwiw, I feel bad about this
	localProjects[id] = target

	return target, nil
}

// Fork creates a user fork of a GitLab project using the specified protocol
func Fork(projID interface{}, opts *gitlab.ForkProjectOptions, useHTTP bool, wait bool) (string, error) {
	var id string

	switch v := projID.(type) {
	case int:
		// If numeric ID, we need the complete name with namespace
		p, err := FindProject(v)
		if err != nil {
			return "", err
		}
		id = p.NameWithNamespace
	case string:
		id = v
		// Check if the ID passed already contains the namespace/path that
		// we need.
		if !strings.Contains(id, "/") {
			// Is it a numeric ID passed as string?
			if _, err := strconv.Atoi(id); err != nil {
				return "", errors.New("remote must include namespace")
			}

			// Do the same as done in 'case int' for numeric ID passed as
			// string
			p, err := FindProject(id)
			if err != nil {
				return "", err
			}
			id = p.NameWithNamespace
		}
	}

	parts := strings.Split(id, "/")

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
		if target.PathWithNamespace == projID {
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
			target.ForkedFromProject.PathWithNamespace != projID {
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

	target, err = FindProject(projID)
	if err != nil {
		return "", err
	}

	// Now that we have the "wait" opt, don't let the user in the hope that
	// something is running.
	fmt.Printf("Forking %s project...\n", projID)
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
func MRCreate(projID interface{}, opts *gitlab.CreateMergeRequestOptions) (string, error) {
	mr, _, err := lab.MergeRequests.CreateMergeRequest(projID, opts)
	if err != nil {
		return "", err
	}
	return mr.WebURL, nil
}

// MRCreateDiscussion creates a discussion on a merge request on GitLab
func MRCreateDiscussion(projID interface{}, id int, opts *gitlab.CreateMergeRequestDiscussionOptions) (string, error) {
	discussion, _, err := lab.Discussions.CreateMergeRequestDiscussion(projID, id, opts)
	if err != nil {
		return "", err
	}

	// Unlike MR, Note has no WebURL property, so we have to create it
	// ourselves from the project, noteable id and note id
	note := discussion.Notes[0]

	p, err := FindProject(projID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/merge_requests/%d#note_%d", p.WebURL, note.NoteableIID, note.ID), nil
}

// MRUpdate edits an merge request on a GitLab project
func MRUpdate(projID interface{}, id int, opts *gitlab.UpdateMergeRequestOptions) (string, error) {
	mr, _, err := lab.MergeRequests.UpdateMergeRequest(projID, id, opts)
	if err != nil {
		return "", err
	}

	return mr.WebURL, nil
}

// MRDelete deletes an merge request on a GitLab project
func MRDelete(projID interface{}, id int) error {
	resp, err := lab.MergeRequests.DeleteMergeRequest(projID, id)
	if resp != nil && resp.StatusCode == http.StatusForbidden {
		return ErrStatusForbidden
	}
	if err != nil {
		return err
	}
	return nil
}

// MRCreateNote adds a note to a merge request on GitLab
func MRCreateNote(projID interface{}, id int, opts *gitlab.CreateMergeRequestNoteOptions) (string, error) {
	note, _, err := lab.Notes.CreateMergeRequestNote(projID, id, opts)
	if err != nil {
		return "", err
	}

	// Unlike MR, Note has no WebURL property, so we have to create it
	// ourselves from the project, noteable id and note id
	p, err := FindProject(projID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/merge_requests/%d#note_%d", p.WebURL, note.NoteableIID, note.ID), nil
}

// MRGet retrieves the merge request from GitLab project
func MRGet(projID interface{}, id int) (*gitlab.MergeRequest, error) {
	mr, _, err := lab.MergeRequests.GetMergeRequest(projID, id, nil)
	if err != nil {
		return nil, err
	}

	return mr, nil
}

// MRList lists the MRs on a GitLab project
func MRList(projID interface{}, opts gitlab.ListProjectMergeRequestsOptions, n int) ([]*gitlab.MergeRequest, error) {
	if n == -1 {
		opts.PerPage = maxItemsPerPage
	}

	list, resp, err := lab.MergeRequests.ListProjectMergeRequests(projID, &opts)
	if err != nil {
		return nil, err
	}

	var ok bool
	if opts.Page, ok = hasNextPage(resp); !ok {
		return list, nil
	}

	for len(list) < n || n == -1 {
		if n != -1 {
			opts.PerPage = n - len(list)
		}
		mrs, resp, err := lab.MergeRequests.ListProjectMergeRequests(projID, &opts)
		if err != nil {
			return nil, err
		}
		list = append(list, mrs...)

		if opts.Page, ok = hasNextPage(resp); !ok {
			break
		}
	}

	return list, nil
}

// MRClose closes an mr on a GitLab project
func MRClose(projID interface{}, id int) error {
	mr, _, err := lab.MergeRequests.GetMergeRequest(projID, id, nil)
	if err != nil {
		return err
	}
	if mr.State == "closed" {
		return fmt.Errorf("mr already closed")
	}
	_, _, err = lab.MergeRequests.UpdateMergeRequest(projID, int(id), &gitlab.UpdateMergeRequestOptions{
		StateEvent: gitlab.String("close"),
	})
	if err != nil {
		return err
	}
	return nil
}

// MRReopen reopen an already close mr on a GitLab project
func MRReopen(projID interface{}, id int) error {
	mr, _, err := lab.MergeRequests.GetMergeRequest(projID, id, nil)
	if err != nil {
		return err
	}
	if mr.State == "opened" {
		return fmt.Errorf("mr not closed")
	}
	_, _, err = lab.MergeRequests.UpdateMergeRequest(projID, int(id), &gitlab.UpdateMergeRequestOptions{
		StateEvent: gitlab.String("reopen"),
	})
	if err != nil {
		return err
	}
	return nil
}

// MRListDiscussions retrieves the discussions (aka notes & comments) for a merge request
func MRListDiscussions(projID interface{}, id int) ([]*gitlab.Discussion, error) {
	discussions := []*gitlab.Discussion{}
	opt := &gitlab.ListMergeRequestDiscussionsOptions{
		// 100 is the maximum allowed by the API
		PerPage: maxItemsPerPage,
	}

	for {
		// get a page of discussions from the API ...
		d, resp, err := lab.Discussions.ListMergeRequestDiscussions(projID, id, opt)
		if err != nil {
			return nil, err
		}

		// ... and add them to our collection of discussions
		discussions = append(discussions, d...)

		// if we've seen all the pages, then we can break here.
		// otherwise, update the page number to get the next page.
		var ok bool
		if opt.Page, ok = hasNextPage(resp); !ok {
			break
		}
	}

	return discussions, nil
}

// MRRebase merges an mr on a GitLab project
func MRRebase(projID interface{}, id int) error {
	_, err := lab.MergeRequests.RebaseMergeRequest(projID, int(id))
	if err != nil {
		return err
	}
	return nil
}

// MRMerge merges an mr on a GitLab project
func MRMerge(projID interface{}, id int, opts *gitlab.AcceptMergeRequestOptions) error {
	_, _, err := lab.MergeRequests.AcceptMergeRequest(projID, int(id), opts)
	if err != nil {
		return err
	}
	return nil
}

// MRApprove approves an mr on a GitLab project
func MRApprove(projID interface{}, id int) error {
	_, resp, err := lab.MergeRequestApprovals.ApproveMergeRequest(projID, id, &gitlab.ApproveMergeRequestOptions{})
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
func MRUnapprove(projID interface{}, id int) error {
	resp, err := lab.MergeRequestApprovals.UnapproveMergeRequest(projID, id, nil)
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
func MRSubscribe(projID interface{}, id int) error {
	_, resp, err := lab.MergeRequests.SubscribeToMergeRequest(projID, id, nil)
	if resp != nil && resp.StatusCode == http.StatusNotModified {
		return errors.New("Already subscribed")
	}
	if err != nil {
		return err
	}
	return nil
}

// MRUnsubscribe unsubscribes from a previously mr on a GitLab project
func MRUnsubscribe(projID interface{}, id int) error {
	_, resp, err := lab.MergeRequests.UnsubscribeFromMergeRequest(projID, id, nil)
	if resp != nil && resp.StatusCode == http.StatusNotModified {
		return errors.New("Not subscribed")
	}
	if err != nil {
		return err
	}
	return nil
}

// MRThumbUp places a thumb up/down on a merge request
func MRThumbUp(projID interface{}, id int) error {
	_, _, err := lab.AwardEmoji.CreateMergeRequestAwardEmoji(projID, id, &gitlab.CreateAwardEmojiOptions{
		Name: "thumbsup",
	})
	if err != nil {
		return err
	}
	return nil
}

// MRThumbDown places a thumb up/down on a merge request
func MRThumbDown(projID interface{}, id int) error {
	_, _, err := lab.AwardEmoji.CreateMergeRequestAwardEmoji(projID, id, &gitlab.CreateAwardEmojiOptions{
		Name: "thumbsdown",
	})
	if err != nil {
		return err
	}
	return nil
}

// IssueCreate opens a new issue on a GitLab project
func IssueCreate(projID interface{}, opts *gitlab.CreateIssueOptions) (string, error) {
	mr, _, err := lab.Issues.CreateIssue(projID, opts)
	if err != nil {
		return "", err
	}
	return mr.WebURL, nil
}

// IssueUpdate edits an issue on a GitLab project
func IssueUpdate(projID interface{}, id int, opts *gitlab.UpdateIssueOptions) (string, error) {
	issue, _, err := lab.Issues.UpdateIssue(projID, id, opts)
	if err != nil {
		return "", err
	}
	return issue.WebURL, nil
}

// IssueCreateNote creates a new note on an issue and returns the note URL
func IssueCreateNote(projID interface{}, id int, opts *gitlab.CreateIssueNoteOptions) (string, error) {
	note, _, err := lab.Notes.CreateIssueNote(projID, id, opts)
	if err != nil {
		return "", err
	}

	// Unlike Issue, Note has no WebURL property, so we have to create it
	// ourselves from the project, noteable id and note id
	p, err := FindProject(projID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/issues/%d#note_%d", p.WebURL, note.NoteableIID, note.ID), nil
}

// IssueGet retrieves the issue information from a GitLab project
func IssueGet(projID interface{}, id int) (*gitlab.Issue, error) {
	issue, _, err := lab.Issues.GetIssue(projID, id)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

// IssueList gets a list of issues on a GitLab Project
func IssueList(projID interface{}, opts gitlab.ListProjectIssuesOptions, n int) ([]*gitlab.Issue, error) {
	if n == -1 {
		opts.PerPage = maxItemsPerPage
	}

	list, resp, err := lab.Issues.ListProjectIssues(projID, &opts)
	if err != nil {
		return nil, err
	}

	var ok bool
	if opts.Page, ok = hasNextPage(resp); !ok {
		return list, nil
	}

	opts.Page = resp.NextPage
	for len(list) < n || n == -1 {
		if n != -1 {
			opts.PerPage = n - len(list)
		}
		issues, resp, err := lab.Issues.ListProjectIssues(projID, &opts)
		if err != nil {
			return nil, err
		}
		list = append(list, issues...)

		if opts.Page, ok = hasNextPage(resp); !ok {
			break
		}
	}
	return list, nil
}

// IssueClose closes an issue on a GitLab project
func IssueClose(projID interface{}, id int) error {
	issue, _, err := lab.Issues.GetIssue(projID, id)
	if err != nil {
		return err
	}
	if issue.State == "closed" {
		return fmt.Errorf("issue already closed")
	}
	_, _, err = lab.Issues.UpdateIssue(projID, id, &gitlab.UpdateIssueOptions{
		StateEvent: gitlab.String("close"),
	})
	if err != nil {
		return err
	}
	return nil
}

// IssueDuplicate closes an issue as duplicate of another
func IssueDuplicate(projID interface{}, id int, dupID interface{}) error {
	dID, err := parseID(dupID)
	if err != nil {
		return err
	}

	// Not exposed in API, go through quick action
	body := "/duplicate " + dID

	_, _, err = lab.Notes.CreateIssueNote(projID, id, &gitlab.CreateIssueNoteOptions{
		Body: &body,
	})
	if err != nil {
		return errors.Errorf("Failed to close issue #%d as duplicate of %s", id, dID)
	}

	issue, _, err := lab.Issues.GetIssue(projID, id)
	if issue == nil || issue.State != "closed" {
		return errors.Errorf("Failed to close issue #%d as duplicate of %s", id, dID)
	}
	return nil
}

// IssueReopen reopens a closed issue
func IssueReopen(projID interface{}, id int) error {
	issue, _, err := lab.Issues.GetIssue(projID, id)
	if err != nil {
		return err
	}
	if issue.State == "opened" {
		return fmt.Errorf("issue not closed")
	}
	_, _, err = lab.Issues.UpdateIssue(projID, id, &gitlab.UpdateIssueOptions{
		StateEvent: gitlab.String("reopen"),
	})
	if err != nil {
		return err
	}
	return nil
}

// IssueListDiscussions retrieves the discussions (aka notes & comments) for an issue
func IssueListDiscussions(projID interface{}, id int) ([]*gitlab.Discussion, error) {
	discussions := []*gitlab.Discussion{}
	opt := &gitlab.ListIssueDiscussionsOptions{
		// 100 is the maximum allowed by the API
		PerPage: maxItemsPerPage,
	}

	for {
		// get a page of discussions from the API ...
		d, resp, err := lab.Discussions.ListIssueDiscussions(projID, id, opt)
		if err != nil {
			return nil, err
		}

		// ... and add them to our collection of discussions
		discussions = append(discussions, d...)

		// if we've seen all the pages, then we can break here.
		// otherwise, update the page number to get the next page.
		var ok bool
		if opt.Page, ok = hasNextPage(resp); !ok {
			break
		}
	}

	return discussions, nil
}

// IssueSubscribe subscribes to an issue on a GitLab project
func IssueSubscribe(projID interface{}, id int) error {
	_, resp, err := lab.Issues.SubscribeToIssue(projID, id, nil)
	if resp != nil && resp.StatusCode == http.StatusNotModified {
		return errors.New("Already subscribed")
	}
	if err != nil {
		return err
	}
	return nil
}

// IssueUnsubscribe unsubscribes from an issue on a GitLab project
func IssueUnsubscribe(projID interface{}, id int) error {
	_, resp, err := lab.Issues.UnsubscribeFromIssue(projID, id, nil)
	if resp != nil && resp.StatusCode == http.StatusNotModified {
		return errors.New("Not subscribed")
	}
	if err != nil {
		return err
	}
	return nil
}

// GetCommit returns top Commit by ref (hash, branch or tag).
func GetCommit(projID interface{}, ref string) (*gitlab.Commit, error) {
	c, _, err := lab.Commits.GetCommit(projID, ref)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// LabelList gets a list of labels on a GitLab Project
func LabelList(projID interface{}) ([]*gitlab.Label, error) {
	labels := []*gitlab.Label{}
	opt := &gitlab.ListLabelsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: maxItemsPerPage,
		},
	}

	for {
		l, resp, err := lab.Labels.ListLabels(projID, opt)
		if err != nil {
			return nil, err
		}

		labels = append(labels, l...)

		// if we've seen all the pages, then we can break here
		// otherwise, update the page number to get the next page.
		var ok bool
		if opt.Page, ok = hasNextPage(resp); !ok {
			break
		}
	}

	return labels, nil
}

// LabelCreate creates a new project label
func LabelCreate(projID interface{}, opts *gitlab.CreateLabelOptions) error {
	_, _, err := lab.Labels.CreateLabel(projID, opts)
	return err
}

// LabelDelete removes a project label
func LabelDelete(projID, name string) error {
	_, err := lab.Labels.DeleteLabel(projID, &gitlab.DeleteLabelOptions{
		Name: &name,
	})
	return err
}

// BranchList get all branches from the project that somehow matches the
// requested options
func BranchList(projID interface{}, opts *gitlab.ListBranchesOptions) ([]*gitlab.Branch, error) {
	branches := []*gitlab.Branch{}
	for {
		bList, resp, err := lab.Branches.ListBranches(projID, opts)
		if err != nil {
			return nil, err
		}
		branches = append(branches, bList...)

		var ok bool
		if opts.Page, ok = hasNextPage(resp); !ok {
			break
		}
	}

	return branches, nil
}

// MilestoneGet get a specific milestone from the list of available ones
func MilestoneGet(projID interface{}, name string) (*gitlab.Milestone, error) {
	opts := &gitlab.ListMilestonesOptions{
		Search: &name,
	}
	milestones, _ := MilestoneList(projID, opts)

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
func MilestoneList(projID interface{}, opt *gitlab.ListMilestonesOptions) ([]*gitlab.Milestone, error) {
	milestones := []*gitlab.Milestone{}
	for {
		m, resp, err := lab.Milestones.ListMilestones(projID, opt)
		if err != nil {
			return nil, err
		}

		milestones = append(milestones, m...)

		// if we've seen all the pages, then we can break here.
		// otherwise, update the page number to get the next page.
		var ok bool
		if opt.Page, ok = hasNextPage(resp); !ok {
			break
		}
	}

	p, err := FindProject(projID)
	if err != nil {
		return nil, err
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
		// otherwise, update the page number to get the next page.
		var ok bool
		if gopt.Page, ok = hasNextPage(resp); !ok {
			break
		}
	}

	return milestones, nil
}

// MilestoneCreate creates a new project milestone
func MilestoneCreate(projID interface{}, opts *gitlab.CreateMilestoneOptions) error {
	_, _, err := lab.Milestones.CreateMilestone(projID, opts)
	return err
}

// MilestoneDelete deletes a project milestone
func MilestoneDelete(projID, name string) error {
	milestone, err := MilestoneGet(projID, name)
	if err != nil {
		return err
	}

	_, err = lab.Milestones.DeleteMilestone(milestone.ProjectID, milestone.ID)
	return err
}

// ProjectSnippetCreate creates a snippet in a project
func ProjectSnippetCreate(projID interface{}, opts *gitlab.CreateProjectSnippetOptions) (*gitlab.Snippet, error) {
	snip, _, err := lab.ProjectSnippets.CreateSnippet(projID, opts)
	if err != nil {
		return nil, err
	}

	return snip, nil
}

// ProjectSnippetDelete deletes a project snippet
func ProjectSnippetDelete(projID interface{}, id int) error {
	_, err := lab.ProjectSnippets.DeleteSnippet(projID, id)
	return err
}

// ProjectSnippetList lists snippets on a project
func ProjectSnippetList(projID interface{}, opts gitlab.ListProjectSnippetsOptions, n int) ([]*gitlab.Snippet, error) {
	if n == -1 {
		opts.PerPage = maxItemsPerPage
	}
	list, resp, err := lab.ProjectSnippets.ListSnippets(projID, &opts)
	if err != nil {
		return nil, err
	}

	var ok bool
	if opts.Page, ok = hasNextPage(resp); !ok {
		return list, nil
	}

	for len(list) < n || n == -1 {
		if n != -1 {
			opts.PerPage = n - len(list)
		}
		snips, resp, err := lab.ProjectSnippets.ListSnippets(projID, &opts)
		if err != nil {
			return nil, err
		}
		list = append(list, snips...)

		if opts.Page, ok = hasNextPage(resp); !ok {
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

	var ok bool
	if opts.Page, ok = hasNextPage(resp); !ok {
		return list, nil
	}

	for len(list) < n || n == -1 {
		if n != -1 {
			opts.PerPage = n - len(list)
		}
		snips, resp, err := lab.Snippets.ListSnippets(&opts)
		if err != nil {
			return nil, err
		}
		list = append(list, snips...)

		if opts.Page, ok = hasNextPage(resp); !ok {
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
func ProjectDelete(projID interface{}) error {
	_, err := lab.Projects.DeleteProject(projID)
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

	var ok bool
	if opts.Page, ok = hasNextPage(resp); !ok {
		return list, nil
	}

	for len(list) < n || n == -1 {
		if n != -1 {
			opts.PerPage = n - len(list)
		}
		projects, resp, err := lab.Projects.ListProjects(&opts)
		if err != nil {
			return nil, err
		}
		list = append(list, projects...)

		if opts.Page, ok = hasNextPage(resp); !ok {
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
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("invalid group query")
	}

	groups := strings.Split(query, "/")
	list, _, err := lab.Groups.SearchGroup(groups[len(groups)-1])
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("group '%s' not found", query)
	}
	if len(list) == 1 {
		return list[0], nil
	}

	for _, group := range list {
		fullName := strings.TrimSpace(group.FullName)
		if group.FullPath == query || fullName == query {
			return group, nil
		}
	}

	msg := fmt.Sprintf("found multiple groups with ambiguous name:\n")
	for _, group := range list {
		msg += fmt.Sprintf("\t%s\n", group.FullPath)
	}
	msg += fmt.Sprintf("use one of the above path options\n")

	return nil, errors.New(msg)
}

// CIJobs returns a list of jobs in the pipeline with given id.
// This function by default doesn't follow bridge jobs.
// The jobs are returned sorted by their CreatedAt time
func CIJobs(projID interface{}, id int, followBridge bool, bridgeName string) ([]JobStruct, error) {
	opts := &gitlab.ListJobsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: maxItemsPerPage,
		},
	}

	// First we get the jobs with direct relation to the actual project
	list := make([]JobStruct, 0)
	var ok bool

	for {
		jobs, resp, err := lab.Jobs.ListPipelineJobs(projID, id, opts)
		if err != nil {
			return nil, err
		}

		for _, job := range jobs {
			list = append(list, JobStruct{job, projID})
		}

		if opts.Page, ok = hasNextPage(resp); !ok {
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
			bridges, resp, err := lab.Jobs.ListPipelineBridges(projID, id, opts)
			if err != nil {
				return nil, err
			}
			bridgeList = append(bridgeList, bridges...)

			if opts.Page, ok = hasNextPage(resp); !ok {
				break
			}
		}

		for _, bridge := range bridgeList {
			if bridgeName != "" && bridge.Name != bridgeName {
				continue
			}

			// Switch to the new project name and downstream pipeline id
			projID = bridge.DownstreamPipeline.ProjectID
			id = bridge.DownstreamPipeline.ID

			for {
				// Get the list of bridged jobs and append to the original list
				jobs, resp, err := lab.Jobs.ListPipelineJobs(projID, id, opts)
				if err != nil {
					return nil, err
				}

				for _, job := range jobs {
					list = append(list, JobStruct{job, projID})
				}

				if opts.Page, ok = hasNextPage(resp); !ok {
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
func CITrace(projID interface{}, id int, name string, followBridge bool, bridgeName string) (io.Reader, *gitlab.Job, error) {
	jobs, err := CIJobs(projID, id, followBridge, bridgeName)
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
		projID = jobStruct.ProjectID
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

	r, _, err := lab.Jobs.GetTraceFile(projID, job.ID)
	if err != nil {
		return nil, job, err
	}

	return r, job, err
}

// CIArtifacts searches by name for a job and returns its artifacts archive
// together with the upstream filename. If path is specified and refers to
// a single file within the artifacts archive, that file is returned instead.
// If no name is provided, the last job with an artifacts file is picked.
func CIArtifacts(projID interface{}, id int, name, path string, followBridge bool, bridgeName string) (io.Reader, string, error) {
	jobs, err := CIJobs(projID, id, followBridge, bridgeName)
	if len(jobs) == 0 || err != nil {
		return nil, "", err
	}
	var (
		job               *gitlab.Job
		lastWithArtifacts *gitlab.Job
	)

	for _, jobStruct := range jobs {
		// Switch to the project ID that owns the job (for a bridge case)
		projID = jobStruct.ProjectID
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
		r, _, err = lab.Jobs.DownloadSingleArtifactsFile(projID, job.ID, path, nil)
		outpath = filepath.Base(path)
	} else {
		r, _, err = lab.Jobs.GetJobArtifacts(projID, job.ID, nil)
		outpath = job.ArtifactsFile.Filename
	}

	if err != nil {
		return nil, "", err
	}

	return r, outpath, nil
}

// CIPlayOrRetry runs a job either by playing it for the first time or by
// retrying it based on the currently known job state
func CIPlayOrRetry(projID interface{}, jobID int, status string) (*gitlab.Job, error) {
	switch status {
	case "pending", "running":
		return nil, nil
	case "manual":
		j, _, err := lab.Jobs.PlayJob(projID, jobID)
		if err != nil {
			return nil, err
		}
		return j, nil
	default:

		j, _, err := lab.Jobs.RetryJob(projID, jobID)
		if err != nil {
			return nil, err
		}

		return j, nil
	}
}

// CICancel cancels a job for a given project by its ID.
func CICancel(projID interface{}, jobID int) (*gitlab.Job, error) {
	j, _, err := lab.Jobs.CancelJob(projID, jobID)
	if err != nil {
		return nil, err
	}
	return j, nil
}

// CICreate creates a pipeline for given ref
func CICreate(projID interface{}, opts *gitlab.CreatePipelineOptions) (*gitlab.Pipeline, error) {
	p, _, err := lab.Pipelines.CreatePipeline(projID, opts)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// CITrigger triggers a pipeline for given ref
func CITrigger(projID interface{}, opts gitlab.RunPipelineTriggerOptions) (*gitlab.Pipeline, error) {
	p, _, err := lab.PipelineTriggers.RunPipelineTrigger(projID, &opts)
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
func AddMRDiscussionNote(projID interface{}, mrID int, discussionID string, body string) (string, error) {
	opts := &gitlab.AddMergeRequestDiscussionNoteOptions{
		Body: &body,
	}

	note, _, err := lab.Discussions.AddMergeRequestDiscussionNote(projID, mrID, discussionID, opts)
	if err != nil {
		return "", err
	}

	p, err := FindProject(projID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/merge_requests/%d#note_%d", p.WebURL, note.NoteableIID, note.ID), nil
}

// AddIssueDiscussionNote adds a note to an existing issue discussion on GitLab
func AddIssueDiscussionNote(projID interface{}, issueID int, discussionID string, body string) (string, error) {
	opts := &gitlab.AddIssueDiscussionNoteOptions{
		Body: &body,
	}

	note, _, err := lab.Discussions.AddIssueDiscussionNote(projID, issueID, discussionID, opts)
	if err != nil {
		return "", err
	}

	p, err := FindProject(projID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/issues/%d#note_%d", p.WebURL, note.NoteableIID, note.ID), nil
}

// UpdateIssueDiscussionNote updates a specific discussion or note in the
// specified issue number
func UpdateIssueDiscussionNote(projID interface{}, issueID int, discussionID string, noteID int, body string) (string, error) {
	opts := &gitlab.UpdateIssueDiscussionNoteOptions{
		Body: &body,
	}

	note, _, err := lab.Discussions.UpdateIssueDiscussionNote(projID, issueID, discussionID, noteID, opts)
	if err != nil {
		return "", err
	}

	p, err := FindProject(projID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/issues/%d#note_%d", p.WebURL, note.NoteableIID, note.ID), nil
}

// UpdateMRDiscussionNote updates a specific discussion or note in the
// specified MR ID.
func UpdateMRDiscussionNote(projID interface{}, mrID int, discussionID string, noteID int, body string) (string, error) {
	opts := &gitlab.UpdateMergeRequestDiscussionNoteOptions{
		Body: &body,
	}

	note, _, err := lab.Discussions.UpdateMergeRequestDiscussionNote(projID, mrID, discussionID, noteID, opts)
	if err != nil {
		return "", err
	}

	p, err := FindProject(projID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/merge_requests/%d#note_%d", p.WebURL, note.NoteableIID, note.ID), nil
}

// ListMRsClosingIssue returns a list of MR IDs that has relation to an issue
// being closed
func ListMRsClosingIssue(projID interface{}, id int) ([]int, error) {
	var retArray []int

	mrs, _, err := lab.Issues.ListMergeRequestsClosingIssue(projID, id, nil, nil)
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
func ListMRsRelatedToIssue(projID interface{}, id int) ([]int, error) {
	var retArray []int

	mrs, _, err := lab.Issues.ListMergeRequestsRelatedToIssue(projID, id, nil, nil)
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
func ListIssuesClosedOnMerge(projID interface{}, id int) ([]int, error) {
	var retArray []int

	issues, _, err := lab.MergeRequests.GetIssuesClosedOnMerge(projID, id, nil, nil)
	if err != nil {
		return retArray, err
	}

	for _, issue := range issues {
		retArray = append(retArray, issue.IID)
	}

	return retArray, nil
}

// MoveIssue moves one issue from one project to another
func MoveIssue(projID interface{}, id int, destProjID interface{}) (string, error) {
	destProject, err := FindProject(destProjID)
	if err != nil {
		return "", err
	}

	opts := &gitlab.MoveIssueOptions{
		ToProjectID: &destProject.ID,
	}

	issue, _, err := lab.Issues.MoveIssue(projID, id, opts)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/issues/%d", destProject.WebURL, issue.IID), nil
}

// GetMRApprovalsConfiguration returns the current MR approval rule
func GetMRApprovalsConfiguration(projID interface{}, id int) (*gitlab.MergeRequestApprovals, error) {
	configuration, _, err := lab.MergeRequestApprovals.GetConfiguration(projID, id)
	if err != nil {
		return nil, err
	}

	return configuration, err
}

// ResolveMRDiscussion resolves a discussion (blocking thread) based on its ID
func ResolveMRDiscussion(projID interface{}, mrID int, discussionID string, noteID int) (string, error) {
	opts := &gitlab.ResolveMergeRequestDiscussionOptions{
		Resolved: gitlab.Bool(true),
	}

	discussion, _, err := lab.Discussions.ResolveMergeRequestDiscussion(projID, mrID, discussionID, opts)
	if err != nil {
		return discussion.ID, err
	}

	p, err := FindProject(projID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Resolved %s/merge_requests/%d#note_%d", p.WebURL, mrID, noteID), nil
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

	var ok bool
	if opts.Page, ok = hasNextPage(resp); !ok {
		return list, nil
	}

	for len(list) < n || n == -1 {
		if n != -1 {
			opts.PerPage = n - len(list)
		}

		todos, resp, err := lab.Todos.ListTodos(&opts)
		if err != nil {
			return nil, err
		}
		list = append(list, todos...)

		if opts.Page, ok = hasNextPage(resp); !ok {
			break
		}
	}

	return list, nil
}

// TodoMarkDone marks a specific Todo as done
func TodoMarkDone(id int) error {
	_, err := lab.Todos.MarkTodoAsDone(id)
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
func TodoMRCreate(projID interface{}, id int) (int, error) {
	todo, resp, err := lab.MergeRequests.CreateTodo(projID, id)
	if err != nil {
		if resp.StatusCode == http.StatusNotModified {
			return 0, ErrNotModified
		}
		return 0, err
	}
	return todo.ID, nil
}

// TodoIssueCreate create a Todo item for an specific Issue
func TodoIssueCreate(projID interface{}, id int) (int, error) {
	todo, resp, err := lab.Issues.CreateTodo(projID, id)
	if err != nil {
		if resp.StatusCode == http.StatusNotModified {
			return 0, ErrNotModified
		}
		return 0, err
	}
	return todo.ID, nil
}

func GetCommitDiff(projID interface{}, sha string) ([]*gitlab.Diff, error) {
	var diffs []*gitlab.Diff
	opt := &gitlab.GetCommitDiffOptions{
		PerPage: maxItemsPerPage,
	}

	for {
		ds, resp, err := lab.Commits.GetCommitDiff(projID, sha, opt)
		if err != nil {
			if resp.StatusCode == 404 {
				log.Fatalf("Cannot find diff for commit %s.  Verify the commit ID or add more characters to the commit ID.", sha)
			}
			return nil, err
		}

		diffs = append(diffs, ds...)

		// if we've seen all the pages, then we can break here
		// otherwise, update the page number to get the next page.
		var ok bool
		if opt.Page, ok = hasNextPage(resp); !ok {
			break
		}
	}

	return diffs, nil
}

func CreateCommitComment(projID interface{}, sha string, newFile string, oldFile string, line int, linetype string, comment string) (string, error) {
	// Ideally want to use lab.Commits.PostCommitComment, however,
	// that API only support comments on linetype=new.
	//
	// https://gitlab.com/gitlab-org/gitlab/-/issues/335337
	commitInfo, err := GetCommit(projID, sha)
	if err != nil {
		fmt.Printf("Could not get diff for commit %s.\n", sha)
		return "", err
	}

	if len(commitInfo.ParentIDs) > 1 {
		log.Fatalf("Commit %s has mulitple parents.  This interface cannot be used for comments.\n", sha)
		return "", err
	}

	position := gitlab.NotePosition{
		BaseSHA:      commitInfo.ParentIDs[0],
		StartSHA:     commitInfo.ParentIDs[0],
		HeadSHA:      sha,
		PositionType: "text",
	}

	switch linetype {
	case "new":
		position.NewPath = newFile
		position.NewLine = line
	case "old":
		position.OldPath = oldFile
		position.OldLine = line
	case "context":
		position.NewPath = newFile
		position.NewLine = line
		position.OldPath = oldFile
		position.OldLine = line
	}

	opt := &gitlab.CreateCommitDiscussionOptions{
		Body:     &comment,
		Position: &position,
	}

	commitDiscussion, _, err := lab.Discussions.CreateCommitDiscussion(projID, sha, opt)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s#note_%d", commitInfo.WebURL, commitDiscussion.Notes[0].ID), nil
}

func CreateMergeRequestCommitDiscussion(projID interface{}, id int, sha string, newFile string, oldFile string, line int, linetype string, comment string) (string, error) {
	commitInfo, err := GetCommit(projID, sha)
	if err != nil {
		fmt.Printf("Could not get diff for commit %s.\n", sha)
		return "", err
	}

	if len(commitInfo.ParentIDs) > 1 {
		log.Fatalf("Commit %s has mulitple parents.  This interface cannot be used for comments.\n", sha)
		return "", err
	}

	position := gitlab.NotePosition{
		NewPath:      newFile,
		OldPath:      oldFile,
		BaseSHA:      commitInfo.ParentIDs[0],
		StartSHA:     commitInfo.ParentIDs[0],
		HeadSHA:      sha,
		PositionType: "text",
	}

	switch linetype {
	case "new":
		position.NewLine = line
	case "old":
		position.OldLine = line
	case "context":
		position.NewLine = line
		position.OldLine = line
	}

	opt := &gitlab.CreateMergeRequestDiscussionOptions{
		Body:     &comment,
		Position: &position,
		CommitID: &sha,
	}

	discussion, _, err := lab.Discussions.CreateMergeRequestDiscussion(projID, id, opt)
	if err != nil {
		return "", err
	}

	note := discussion.Notes[0]
	p, err := FindProject(projID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/merge_requests/%d#note_%d", p.WebURL, note.NoteableIID, note.ID), nil
}

// hasNextPage get the next page number in case the API response has more
// than one. It also uses only the "X-Page" and "X-Next-Page" HTTP headers,
// since in some cases the API response may come without the HTTP
// X-Total(-Page) header. Reference:
// https://docs.gitlab.com/ee/user/gitlab_com/index.html#pagination-response-headers
func hasNextPage(resp *gitlab.Response) (int, bool) {
	if resp.CurrentPage >= resp.NextPage {
		return 0, false
	}
	return resp.NextPage, true
}
