package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "github.com/xanzy/go-gitlab"
)

// issueEditCmdTestCreateIssue creates an issue and returns the issue number
func issueEditCmdTestCreateIssue(t *testing.T, dir string) string {
	cmd := exec.Command(labBinaryPath, "issue", "create", "lab-testing",
		"-m", "issue title", "-l", "bug")
	cmd.Dir = dir

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	s := strings.Split(string(b), "\n")
	s = strings.Split(s[0], "/")
	return s[len(s)-1]
}

// issueEditCmdTestShowIssue returns the `lab issue show` output for the given issue
func issueEditCmdTestShowIssue(t *testing.T, dir string, issueNum string) string {
	cmd := exec.Command(labBinaryPath, "issue", "show", "lab-testing", issueNum)
	cmd.Dir = dir

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	return string(b)
}

func Test_issueEditCmd(t *testing.T) {
	repo := copyTestRepo(t)

	issueNum := issueEditCmdTestCreateIssue(t, repo)

	// update the issue
	cmd := exec.Command(labBinaryPath, "issue", "edit", "lab-testing", issueNum,
		"-m", "new title")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	// show the updated issue
	issueShowOuput := issueEditCmdTestShowIssue(t, repo, issueNum)

	// the output should show the updated title, not the old title
	require.Contains(t, issueShowOuput, "new title")
	require.NotContains(t, issueShowOuput, "issue title")
}

func Test_issueEditLabels(t *testing.T) {
	repo := copyTestRepo(t)

	issueNum := issueEditCmdTestCreateIssue(t, repo)

	// update the issue
	cmd := exec.Command(labBinaryPath, "issue", "edit", "lab-testing", issueNum,
		"-l", "crit", "--unlabel", "bug")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	// show the updated issue
	issueShowOuput := issueEditCmdTestShowIssue(t, repo, issueNum)

	// the output should show the updated labels
	require.Contains(t, issueShowOuput, "critical")
	require.NotContains(t, issueShowOuput, "bug")
}

func Test_issueEditAssignees(t *testing.T) {
	repo := copyTestRepo(t)

	issueNum := issueEditCmdTestCreateIssue(t, repo)

	// add an assignee
	cmd := exec.Command(labBinaryPath, "issue", "edit", "lab-testing", issueNum,
		"-a", "lab-testing")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	// get the updated issue
	issueShowOuput := issueEditCmdTestShowIssue(t, repo, issueNum)

	// the output should show the new assignee
	require.Contains(t, issueShowOuput, "Assignees: lab-testing")

	// now remove the assignee
	cmd = exec.Command(labBinaryPath, "issue", "edit", "lab-testing", issueNum,
		"--unassign", "lab-testing")
	cmd.Dir = repo

	b, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	// get the updated issue again
	issueShowOuput = issueEditCmdTestShowIssue(t, repo, issueNum)

	// the output should NOT show the assignee
	require.NotContains(t, issueShowOuput, "Assignees: lab-testing")
}

func Test_issueEditGetTitleAndDescription(t *testing.T) {
	tests := []struct {
		Name                string
		Issue               *gitlab.Issue
		Args                []string
		ExpectedTitle       string
		ExpectedDescription string
	}{
		{
			Name: "Using messages",
			Issue: &gitlab.Issue{
				Title:       "old title",
				Description: "old body",
			},
			Args:                []string{"new title", "new body 1", "new body 2"},
			ExpectedTitle:       "new title",
			ExpectedDescription: "new body 1\n\nnew body 2",
		},
		{
			Name: "Using a single message",
			Issue: &gitlab.Issue{
				Title:       "old title",
				Description: "old body",
			},
			Args:                []string{"new title"},
			ExpectedTitle:       "new title",
			ExpectedDescription: "old body",
		},
		{
			Name: "From Editor",
			Issue: &gitlab.Issue{
				Title:       "old title",
				Description: "old body",
			},
			Args:                nil,
			ExpectedTitle:       "old title",
			ExpectedDescription: "old body",
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			title, body, err := editGetTitleDescription(test.Issue.Title, test.Issue.Description, test.Args, len(test.Args))
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.ExpectedTitle, title)
			assert.Equal(t, test.ExpectedDescription, body)
		})
	}
}

func Test_issueEditText(t *testing.T) {
	t.Parallel()
	text, err := editText("old title", "old body")
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, `old title

old body

# Edit the title and/or description. The first block of text
# is the title and the rest is the description.`, text)

}

func Test_issueEditSame(t *testing.T) {
	t.Parallel()
	assert.True(t, same([]string{}, []string{}))
	assert.True(t, same([]string{"a"}, []string{"a"}))
	assert.True(t, same([]string{"a", "b"}, []string{"a", "b"}))
	assert.True(t, same([]string{"a", "b"}, []string{"b", "a"}))
	assert.True(t, same([]string{"b", "a"}, []string{"a", "b"}))

	assert.False(t, same([]string{"a"}, []string{}))
	assert.False(t, same([]string{"a"}, []string{"c"}))
	assert.False(t, same([]string{}, []string{"c"}))
	assert.False(t, same([]string{"a", "b"}, []string{"a", "c"}))
	assert.False(t, same([]string{"a", "b"}, []string{"a"}))
	assert.False(t, same([]string{"a", "b"}, []string{"c"}))
}

func Test_issueEditUnion(t *testing.T) {
	t.Parallel()
	s := union([]string{"a", "b"}, []string{"c"})
	assert.Equal(t, 3, len(s))
	assert.True(t, same(s, []string{"a", "b", "c"}))
}

func Test_issueEditDifference(t *testing.T) {
	t.Parallel()
	s := difference([]string{"a", "b"}, []string{"c"})
	assert.Equal(t, 2, len(s))
	assert.True(t, same(s, []string{"a", "b"}))

	s = difference([]string{"a", "b"}, []string{"a", "c"})
	assert.Equal(t, 1, len(s))
	assert.True(t, same(s, []string{"b"}))
}
