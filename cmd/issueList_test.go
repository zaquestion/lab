package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_issueList(t *testing.T) {
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "issue", "list")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	issues := strings.Split(string(b), "\n")
	t.Log(issues)
	firstIssue := issues[len(issues)-2 : len(issues)-1]
	require.Equal(t, "#1 test issue for lab list", firstIssue[0])
}
