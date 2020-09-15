package cmd

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_mrCreateNote(t *testing.T) {
	tests := []struct {
		Name         string
		Args         []string
		ExpectedBody string
	}{
		{
			Name:         "Normal text",
			Args:         []string{"lab-testing", "1", "-m", "note text"},
			ExpectedBody: "https://gitlab.com/lab-testing/test/merge_requests/1#note_",
		},
		{
			// Escaped sequence text direct in the argument list as the
			// following one was already a problem:
			// https://github.com/zaquestion/lab/issues/376
			Name:         "Escape char",
			Args:         []string{"lab-testing", "1", "-m", "{\"key\": \"value\"}"},
			ExpectedBody: "https://gitlab.com/lab-testing/test/merge_requests/1#note_",
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

	cmd := exec.Command(labBinaryPath, "mr", "note", "lab-testing", "1",
		"-F", "hellolab.txt")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	require.Contains(t, string(b), "https://gitlab.com/lab-testing/test/merge_requests/1#note_")
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
			body, err := mrNoteMsg(test.Msgs)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, test.ExpectedBody, body)
		})
	}
}

func Test_mrNoteText(t *testing.T) {
	t.Parallel()
	text, err := mrNoteText()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, `

# Write a message for this note. Commented lines are discarded.`, text)

}
