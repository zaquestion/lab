package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_mrCheckoutCmdRun(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)

	cmd := exec.Command("../lab_bin", "mr", "checkout", "1")
	cmd.Dir = repo
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
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

func Test_mrCheckoutCmd_track(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)

	cmd := exec.Command("../lab_bin", "mr", "checkout", "1", "-t", "-b", "mrtest_track")
	cmd.Dir = repo
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
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

	cmd = exec.Command("git", "remote", "-v")
	cmd.Dir = repo
	gitOut, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	remotes := string(gitOut)
	require.Contains(t, remotes, "zaquestion	git@gitlab.com:zaquestion/test.git")
}

func Test_mrCheckoutCmdRunWithDifferentName(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)

	cmd := exec.Command("../lab_bin", "mr", "checkout", "1", "-b", "mrtest_custom_name")
	cmd.Dir = repo
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	t.Log(string(b))

	cmd = exec.Command("git", "branch")
	cmd.Dir = repo

	branch, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	require.Contains(t, string(branch), "mrtest_custom_name")

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
