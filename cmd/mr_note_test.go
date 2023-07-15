package cmd

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #3952 is not special, it's just a place to dump discussions as mr #1 filled up, long term should update the tests clean up what they create
const mrCommentSlashDiscussionDumpsterID = "3954"

func Test_mrCreateNote(t *testing.T) {
	tests := []struct {
		Name         string
		Args         []string
		ExpectedBody string
	}{
		{
			Name:         "Normal text",
			Args:         []string{"lab-testing", mrCommentSlashDiscussionDumpsterID, "-m", "note text"},
			ExpectedBody: "https://gitlab.com/lab-testing/test/-/merge_requests/" + mrCommentSlashDiscussionDumpsterID + "#note_",
		},
		{
			// Escaped sequence text direct in the argument list as the
			// following one was already a problem:
			// https://github.com/zaquestion/lab/issues/376
			Name:         "Escape char",
			Args:         []string{"lab-testing", mrCommentSlashDiscussionDumpsterID, "-m", "{\"key\": \"value\"}"},
			ExpectedBody: "https://gitlab.com/lab-testing/test/-/merge_requests/" + mrCommentSlashDiscussionDumpsterID + "#note_",
		},
	}
	noteCmd := []string{"mr", "note"}
	repo := copyTestRepo(t)

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			cmd := exec.Command(labBinaryPath, append(noteCmd, test.Args...)...)
			cmd.Dir = repo

			b, err := cmd.CombinedOutput()
			if err != nil {
				t.Log(string(b))
				t.Fatal(err)
			}

			require.Contains(t, string(b), test.ExpectedBody)
		})
	}
}

func Test_mrCreateNote_file(t *testing.T) {
	repo := copyTestRepo(t)

	err := ioutil.WriteFile(filepath.Join(repo, "hellolab.txt"), []byte("hello\nlab\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(labBinaryPath, "mr", "note", "lab-testing", mrCommentSlashDiscussionDumpsterID,
		"-F", "hellolab.txt")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	require.Contains(t, string(b), "https://gitlab.com/lab-testing/test/-/merge_requests/"+mrCommentSlashDiscussionDumpsterID+"#note_")
}

func Test_mrReplyAndResolve(t *testing.T) {
	repo := copyTestRepo(t)

	cmd := exec.Command(labBinaryPath, "mr", "note", "lab-testing", mrCommentSlashDiscussionDumpsterID, "-m", "merge request text")
	cmd.Dir = repo

	a, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(a))
		t.Fatal(err)
	}
	_noteIDs := strings.Split(string(a), "\n")
	noteIDs := strings.Split(_noteIDs[0], "#note_")
	noteID := noteIDs[1]

	// add reply to the noteID
	reply := exec.Command(labBinaryPath, "mr", "reply", "lab-testing", mrCommentSlashDiscussionDumpsterID+":"+noteID, "-m", "reply to note")
	reply.Dir = repo
	c, err := reply.CombinedOutput()
	if err != nil {
		t.Log(string(c))
		t.Fatal(err)
	}
	_replyIDs := strings.Split(string(c), "\n")
	replyIDs := strings.Split(_replyIDs[0], "#note_")
	replyID := replyIDs[1]

	show := exec.Command(labBinaryPath, "mr", "show", "lab-testing", mrCommentSlashDiscussionDumpsterID, "--comments")
	show.Dir = repo
	d, err := show.CombinedOutput()
	if err != nil {
		t.Log(string(d))
		t.Fatal(err)
	}

	resolve := exec.Command(labBinaryPath, "mr", "resolve", "lab-testing", mrCommentSlashDiscussionDumpsterID+":"+noteID)
	resolve.Dir = repo
	e, err := resolve.CombinedOutput()
	if err != nil {
		t.Log(string(e))
		t.Fatal(err)
	}

	require.Contains(t, string(d), "#"+noteID+": "+"lab-testing started a discussion")
	require.Contains(t, string(d), "#"+replyID+": "+"lab-testing commented at")
	require.Contains(t, string(e), "Resolved")
}

func Test_mrNoteMsg(t *testing.T) {
	tests := []struct {
		Name         string
		Msgs         []string
		ExpectedBody string
	}{
		{
			Name:         "Using messages",
			Msgs:         []string{"note paragraph 1", "note paragraph 2"},
			ExpectedBody: "note paragraph 1\n\nnote paragraph 2",
		},
		{
			Name:         "From Editor",
			Msgs:         nil,
			ExpectedBody: "", // this is not a great test
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			body, err := noteMsg(test.Msgs, true, 1, "OPEN", "", "\n")
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, test.ExpectedBody, body)
		})
	}
}
