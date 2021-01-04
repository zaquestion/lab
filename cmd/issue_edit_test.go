package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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
