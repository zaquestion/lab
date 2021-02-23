package cmd

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_mrCreateDiscussion(t *testing.T) {
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "discussion", "lab-testing", mrCommentSlashDiscussionDumpsterID,
		"-m", "discussion text")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	require.Contains(t, string(b), "https://gitlab.com/lab-testing/test/merge_requests/"+mrCommentSlashDiscussionDumpsterID+"#note_")
}

func Test_mrCreateDiscussion_file(t *testing.T) {
	repo := copyTestRepo(t)

	err := ioutil.WriteFile(filepath.Join(repo, "hellolab.txt"), []byte("hello\nlab\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(labBinaryPath, "mr", "discussion", "lab-testing", mrCommentSlashDiscussionDumpsterID,
		"-F", "hellolab.txt")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	require.Contains(t, string(b), "https://gitlab.com/lab-testing/test/merge_requests/"+mrCommentSlashDiscussionDumpsterID+"#note_")
}

func Test_mrDiscussionMsg(t *testing.T) {
	tests := []struct {
		Name         string
		Msgs         []string
		ExpectedBody string
	}{
		{
			Name:         "Using messages",
			Msgs:         []string{"discussion paragraph 1", "discussion paragraph 2"},
			ExpectedBody: "discussion paragraph 1\n\ndiscussion paragraph 2",
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
			body, err := mrDiscussionMsg(test.Msgs)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, test.ExpectedBody, body)
		})
	}
}

func Test_mrDiscussionText(t *testing.T) {
	t.Parallel()
	text, err := mrDiscussionText()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, `

# Write a message for this discussion. Commented lines are discarded.`, text)

}
