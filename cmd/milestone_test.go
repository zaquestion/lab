package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_milestoneCmd(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	t.Run("prepare", func(t *testing.T) {
		cmd := exec.Command("sh", "-c", labBinaryPath+` milestone list lab-testing | grep -q 'test-milestone' && `+labBinaryPath+` milestone delete test-milestone`)
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			//t.Fatal(err)
		}
	})
	t.Run("create", func(t *testing.T) {
		cmd := exec.Command(labBinaryPath, "milestone", "create", "lab-testing", "test-milestone")
		cmd.Dir = repo

		b, _ := cmd.CombinedOutput()
		if strings.Contains(string(b), "403 Forbidden") {
			t.Skip("No permission to change milestones, skipping")
		}

		cmd = exec.Command(labBinaryPath, "milestone", "list", "lab-testing")
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		out := string(b)
		if err != nil {
			t.Log(out)
			//t.Fatal(err)
		}
		require.Contains(t, out, "test-milestone")
	})
	t.Run("delete", func(t *testing.T) {
		cmd := exec.Command(labBinaryPath, "milestone", "delete", "lab-testing", "test-milestone")
		cmd.Dir = repo

		_ = cmd.Run()

		cmd = exec.Command(labBinaryPath, "milestone", "list", "lab-testing")
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		out := string(b)
		if err != nil {
			t.Log(out)
			//t.Fatal(err)
		}
		require.NotContains(t, out, "test-milestone")
	})
}
