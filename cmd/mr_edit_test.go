package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func MREditCmdTestCreateMR(t *testing.T, dir string) string {
	git := exec.Command("git", "checkout", "-b", "local/mrtest", "origin/mrtest")
	git.Dir = dir
	g, err := git.CombinedOutput()
	if err != nil {
		t.Log(string(g))
		t.Fatal(err)
	}

	cmd := exec.Command(labBinaryPath, "mr", "create", "lab-testing", "-m", "mr title", "-m", "mr description")
	cmd.Dir = dir

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	i := strings.Index(string(b), "/diffs\n")
	mrID := strings.TrimPrefix(string(b)[:i], "https://gitlab.com/lab-testing/test/-/merge_requests/")
	return mrID
}

func Test_MRNoteDelete(t *testing.T) {
	repo := copyTestRepo(t)

	mrNum := MREditCmdTestCreateMR(t, repo)

	// add just a note "DELETED 1"
	cmd := exec.Command(labBinaryPath, "mr", "note", "lab-testing", mrNum, "-m", "DELETED 1")
	cmd.Dir = repo

	weburl, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(weburl))
		t.Fatal(err)
	}
	noteIDs := strings.Split(string(weburl), "\n")
	noteID := strings.Split(noteIDs[0], "#note_")[1]
	deletedNote := mrNum + ":" + noteID

	// add another note "REPLY 2"
	cmd = exec.Command(labBinaryPath, "mr", "note", "lab-testing", mrNum, "-m", "REPLY 2")
	cmd.Dir = repo

	weburl, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(weburl))
		t.Fatal(err)
	}
	noteIDs = strings.Split(string(weburl), "\n")
	noteID = strings.Split(noteIDs[0], "#note_")[1]
	replyNote := mrNum + ":" + noteID

	// reply to the "REPLY 2" comment with a note to create a discussion "DELETED 2"
	cmd = exec.Command(labBinaryPath, "mr", "reply", "lab-testing", replyNote, "-m", "DELETED 2")
	cmd.Dir = repo

	weburl, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(weburl))
		t.Fatal(err)
	}
	noteIDs = strings.Split(string(weburl), "\n")
	noteID = strings.Split(noteIDs[0], "#note_")[1]
	deletedDiscussion := mrNum + ":" + noteID

	// reply to the comment with a second comment "DISCUSSION 1"
	cmd = exec.Command(labBinaryPath, "mr", "reply", "lab-testing", replyNote, "-m", "DISCUSSION 1")
	cmd.Dir = repo

	weburl, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(weburl))
		t.Fatal(err)
	}

	// delete the first note
	cmd = exec.Command(labBinaryPath, "mr", "edit", "lab-testing", deletedNote, "--delete-note")
	cmd.Dir = repo

	weburl, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(weburl))
		t.Fatal(err)
	}

	// delete the first discussion reply
	cmd = exec.Command(labBinaryPath, "mr", "edit", "lab-testing", deletedDiscussion, "--delete-note")
	cmd.Dir = repo

	weburl, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(weburl))
		t.Fatal(err)
	}

	// show the updated mr and comments
	cmd = exec.Command(labBinaryPath, "mr", "show", "lab-testing", mrNum, "--comments")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	mrShowOutput := string(b)

	git := exec.Command(labBinaryPath, "mr", "close", "lab-testing", mrNum)
	git.Dir = repo
	g, err := git.CombinedOutput()
	if err != nil {
		t.Log(string(g))
		t.Fatal(err)
	}

	// lab show should not contain "DELETED" notes
	require.NotContains(t, mrShowOutput, "DELETED 1")
	require.NotContains(t, mrShowOutput, "DELETED 2")
	// lab show should contain the other notes and disucssion
	require.Contains(t, mrShowOutput, "REPLY 2")
	require.Contains(t, mrShowOutput, "DISCUSSION 1")
}
