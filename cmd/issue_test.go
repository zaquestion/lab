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
			"-m", "issue title")
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
