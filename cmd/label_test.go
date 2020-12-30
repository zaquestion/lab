package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_labelCmd(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	t.Run("prepare", func(t *testing.T) {
		cmd := exec.Command("sh", "-c", labBinaryPath+` label list lab-testing | grep -q 'test-label' && `+labBinaryPath+` label delete test-label`)
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			//t.Fatal(err)
		}
	})
	t.Run("create", func(t *testing.T) {
		cmd := exec.Command(labBinaryPath, "label", "create", "lab-testing", "test-label",
			"--color", "crimson",
			"--description", "Reddish test label",
		)
		cmd.Dir = repo

		_ = cmd.Run()

		cmd = exec.Command(labBinaryPath, "label", "list", "lab-testing")
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		out := string(b)
		if err != nil {
			t.Log(out)
			//t.Fatal(err)
		}
		require.Contains(t, out, "test-label")
	})
	t.Run("delete", func(t *testing.T) {
		cmd := exec.Command(labBinaryPath, "label", "delete", "lab-testing", "test-label")
		cmd.Dir = repo

		_ = cmd.Run()

		cmd = exec.Command(labBinaryPath, "label", "list", "lab-testing")
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		out := string(b)
		if err != nil {
			t.Log(out)
			//t.Fatal(err)
		}
		require.NotContains(t, out, "test-label")
	})
}
