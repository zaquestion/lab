package cmd

import (
	"os/exec"
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
			body, err := noteMsg(test.Msgs)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, test.ExpectedBody, body)
		})
	}
}

func Test_issueNoteText(t *testing.T) {
	t.Parallel()
	text, err := noteText()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, `

# Write a message for this note. Commented lines are discarded.`, text)

}
