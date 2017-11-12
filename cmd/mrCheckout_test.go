package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_mrCheckoutCmdRun(t *testing.T) {
	repo := copyTestRepo(t)

	cmd := exec.Command("../lab_bin", "mr", "checkout", "1")
	cmd.Dir = repo
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))

	cmd = exec.Command("git", "branch")
	cmd.Dir = repo

	branch, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	require.Contains(t, string(branch), "mrtest")

	cmd = exec.Command("git", "log", "-n1")
	cmd.Dir = repo
	log, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	eLog := string(log)
	require.Contains(t, eLog, "Test file for MR test")
	require.Contains(t, eLog, "54fd49a2ac60aeeef5ddc75efecd49f85f7ba9b0")
}
