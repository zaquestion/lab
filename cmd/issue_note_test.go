package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_issueCreateNote(t *testing.T) {
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "issue", "note", "lab-testing", "1",
		"-m", "note text")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	require.Contains(t, string(b), "https://gitlab.com/lab-testing/test/issues/1#note_")
}

func Test_issueReplyNote(t *testing.T) {
	repo := copyTestRepo(t)
	create := exec.Command(labBinaryPath, "issue", "create", "lab-testing", "-m", "note text")
	create.Dir = repo

	a, err := create.CombinedOutput()
	if err != nil {
		t.Log(string(a))
		t.Fatal(err)
	}
	issueIDs := strings.Split(string(a), "\n")
	issueID := strings.Trim(issueIDs[0], "https://gitlab.com/lab-testing/test/-/issues/")

	note := exec.Command(labBinaryPath, "issue", "note", "lab-testing", issueID, "-m", "note text")
	note.Dir = repo

	b, err := note.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	_noteIDs := strings.Split(string(b), "\n")
	noteIDs := strings.Split(_noteIDs[0], "#note_")
	noteID := noteIDs[1]

	// add reply to the noteID
	reply := exec.Command(labBinaryPath, "issue", "reply", "lab-testing", issueID+":"+noteID,
		"-m", "reply to note", "-m", "second reply paragraph")
	reply.Dir = repo
	c, err := reply.CombinedOutput()
	if err != nil {
		t.Log(string(c))
		t.Fatal(err)
	}
	_replyIDs := strings.Split(string(c), "\n")
	replyIDs := strings.Split(_replyIDs[0], "#note_")
	replyID := replyIDs[1]

	show := exec.Command(labBinaryPath, "issue", "show", "lab-testing", issueID, "--comments")
	show.Dir = repo
	d, err := show.CombinedOutput()
	if err != nil {
		t.Log(string(d))
		t.Fatal(err)
	}

	require.Contains(t, string(d), "#"+noteID+": "+"lab-testing started a discussion")
	require.Contains(t, string(d), "#"+replyID+": "+"lab-testing commented at")
	require.Contains(t, string(d), "    reply to note")
	require.Contains(t, string(d), "    second reply paragraph")
}
