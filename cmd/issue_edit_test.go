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

func Test_issueNoteDelete(t *testing.T) {
	repo := copyTestRepo(t)

	issueNum := issueEditCmdTestCreateIssue(t, repo)

	// add just a note "DELETED 1"
	cmd := exec.Command(labBinaryPath, "issue", "note", "lab-testing", issueNum, "-m", "DELETED 1")
	cmd.Dir = repo

	weburl, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(weburl))
		t.Fatal(err)
	}
	noteIDs := strings.Split(string(weburl), "\n")
	noteID := strings.Split(noteIDs[0], "#note_")[1]
	deletedNote := issueNum + ":" + noteID

	// add another note "REPLY 2"
	cmd = exec.Command(labBinaryPath, "issue", "note", "lab-testing", issueNum, "-m", "REPLY 2")
	cmd.Dir = repo

	weburl, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(weburl))
		t.Fatal(err)
	}
	noteIDs = strings.Split(string(weburl), "\n")
	noteID = strings.Split(noteIDs[0], "#note_")[1]
	replyNote := issueNum + ":" + noteID

	// reply to the "REPLY 2" comment with a note to create a discussion "DELETED 2"
	cmd = exec.Command(labBinaryPath, "issue", "reply", "lab-testing", replyNote, "-m", "DELETED 2")
	cmd.Dir = repo

	weburl, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(weburl))
		t.Fatal(err)
	}
	noteIDs = strings.Split(string(weburl), "\n")
	noteID = strings.Split(noteIDs[0], "#note_")[1]
	deletedDiscussion := issueNum + ":" + noteID

	// reply to the comment with a second comment "DISCUSSION 1"
	cmd = exec.Command(labBinaryPath, "issue", "reply", "lab-testing", replyNote, "-m", "DISCUSSION 1")
	cmd.Dir = repo

	weburl, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(weburl))
		t.Fatal(err)
	}

	// delete the first note
	cmd = exec.Command(labBinaryPath, "issue", "edit", "lab-testing", deletedNote, "--delete-note")
	cmd.Dir = repo

	weburl, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(weburl))
		t.Fatal(err)
	}

	// delete the first discussion reply
	cmd = exec.Command(labBinaryPath, "issue", "edit", "lab-testing", deletedDiscussion, "--delete-note")
	cmd.Dir = repo

	weburl, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(weburl))
		t.Fatal(err)
	}

	// show the updated issue and comments
	cmd = exec.Command(labBinaryPath, "issue", "show", "lab-testing", issueNum, "--comments")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	issueShowOutput := string(b)

	// lab show should not contain "DELETED" notes
	require.NotContains(t, issueShowOutput, "DELETED 1")
	require.NotContains(t, issueShowOutput, "DELETED 2")
	// lab show should contain the other notes and disucssion
	require.Contains(t, issueShowOutput, "REPLY 2")
	require.Contains(t, issueShowOutput, "DISCUSSION 1")
}
