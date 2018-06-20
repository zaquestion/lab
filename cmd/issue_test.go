package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_issueCmd(t *testing.T) {
	var issueID string
	t.Run("create", func(t *testing.T) {
		repo := copyTestRepo(t)
		cmd := exec.Command("../lab_bin", "issue", "create", "lab-testing",
			"-m", "issue title",
			"-m", "issue description",
			"-l", "bug",
			"-l", "critical",
			"-a", "lab-testing")
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}

		out := string(b)
		require.Contains(t, out, "https://gitlab.com/lab-testing/test/issues/")

		i := strings.Index(out, "\n")
		issueID = strings.TrimPrefix(out[:i], "https://gitlab.com/lab-testing/test/issues/")
		t.Log(issueID)
	})
	t.Run("show", func(t *testing.T) {
		if issueID == "" {
			t.Skip("issueID is empty, create likely failed")
		}
		repo := copyTestRepo(t)
		cmd := exec.Command("../lab_bin", "issue", "show", "lab-testing", issueID)
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}
		out := string(b)
		require.Contains(t, out, "Project: lab-testing/test\n")
		require.Contains(t, out, "Status: Open\n")
		require.Contains(t, out, "Assignees: lab-testing\n")
		require.Contains(t, out, fmt.Sprintf("#%s issue title", issueID))
		require.Contains(t, out, "===================================\nissue description")
		require.Contains(t, out, "Labels: bug, critical\n")
		require.Contains(t, out, fmt.Sprintf("WebURL: https://gitlab.com/lab-testing/test/issues/%s", issueID))
	})
	t.Run("delete", func(t *testing.T) {
		if issueID == "" {
			t.Skip("issueID is empty, create likely failed")
		}
		repo := copyTestRepo(t)
		cmd := exec.Command("../lab_bin", "issue", "lab-testing", "-d", issueID)
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}
		require.Contains(t, string(b), fmt.Sprintf("Issue #%s closed", issueID))
	})
}

func Test_issueCmd_noArgs(t *testing.T) {
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "issue")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	require.Contains(t, string(b), `Usage:
  lab issue [flags]
  lab issue [command]`)
}
