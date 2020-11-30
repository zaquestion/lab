package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

func Test_issueNoteMsg(t *testing.T) {
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
			body, err := noteMsg(test.Msgs, false, "\n")
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, test.ExpectedBody, body)
		})
	}
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
	reply := exec.Command(labBinaryPath, "issue", "reply", "lab-testing", issueID+":"+noteID, "-m", "reply to note")
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
}

func Test_issueNoteText(t *testing.T) {
	t.Parallel()
	text, err := noteText("\n")
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, `

# Write a message for this note. Commented lines are discarded.`, text)

}
