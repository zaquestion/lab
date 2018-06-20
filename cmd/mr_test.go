package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_mrCmd(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	var mrID string
	t.Run("create", func(t *testing.T) {
		git := exec.Command("git", "checkout", "mrtest")
		git.Dir = repo
		b, err := git.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}

		cmd := exec.Command("../lab_bin", "mr", "create", "lab-testing", "master",
			"-m", "mr title",
			"-m", "mr description",
			"-a", "lab-testing",
		)
		cmd.Dir = repo

		b, _ = cmd.CombinedOutput()
		out := string(b)
		t.Log(out)
		require.Contains(t, out, "https://gitlab.com/lab-testing/test/merge_requests")

		i := strings.Index(out, "/diffs\n")
		mrID = strings.TrimPrefix(out[:i], "https://gitlab.com/lab-testing/test/merge_requests/")
		t.Log(mrID)
	})
	t.Run("show", func(t *testing.T) {
		if mrID == "" {
			t.Skip("mrID is empty, create likely failed")
		}
		cmd := exec.Command("../lab_bin", "mr", "show", "lab-testing", mrID)
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}
		out := string(b)
		require.Contains(t, out, "Project: lab-testing/test\n")
		require.Contains(t, out, "Branches: mrtest->master\n")
		require.Contains(t, out, "Status: Open\n")
		require.Contains(t, out, "Assignee: lab-testing\n")
		require.Contains(t, out, fmt.Sprintf("#%s mr title", mrID))
		require.Contains(t, out, "===================================\nmr description")
		require.Contains(t, out, fmt.Sprintf("WebURL: https://gitlab.com/lab-testing/test/merge_requests/%s", mrID))
	})
	t.Run("delete", func(t *testing.T) {
		if mrID == "" {
			t.Skip("mrID is empty, create likely failed")
		}
		cmd := exec.Command("../lab_bin", "mr", "lab-testing", "-d", mrID)
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}
		require.Contains(t, string(b), fmt.Sprintf("Merge Request #%s closed", mrID))
	})
}

func Test_mrCmd_noArgs(t *testing.T) {
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "mr")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	require.Contains(t, string(b), `Usage:
  lab mr [flags]
  lab mr [command]`)
}
