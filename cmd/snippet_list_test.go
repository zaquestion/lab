package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test for listing personal snippets in snippet_test.go

func Test_snippetList(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "snippet", "list", "-n", "1", "lab-testing")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	require.Regexp(t, `#\d+ snippet title`, string(b))
}
