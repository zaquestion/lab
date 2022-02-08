package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_mrCheckoutCmdRun(t *testing.T) {
	repo := copyTestRepo(t)

	// make sure the branch does not exist
	cmd := exec.Command("git", "branch", "-D", "mrtest")
	cmd.Dir = repo
	cmd.CombinedOutput()

	cmd = exec.Command(labBinaryPath, "mr", "checkout", "1")
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
	repo := copyTestRepo(t)

	// make sure the branch does not exist
	cmd := exec.Command("git", "branch", "-D", "mrtest")
	cmd.Dir = repo
	cmd.CombinedOutput()

	cmd = exec.Command(labBinaryPath, "mr", "checkout", "1", "-f", "-t", "-b", "mrtest_track")
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
	require.Contains(t, remotes, "origin	git@gitlab.com:zaquestion/test.git")
}

func Test_mrCheckoutCmdRunWithDifferentName(t *testing.T) {
	repo := copyTestRepo(t)

	// make sure the branch does not exist
	cmd := exec.Command("git", "branch", "-D", "mrtest")
	cmd.Dir = repo
	cmd.CombinedOutput()

	cmd = exec.Command(labBinaryPath, "mr", "checkout", "1", "-b", "mrtest_custom_name")
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

func Test_mrDoubleCheckoutFailCmdRun(t *testing.T) {
	repo := copyTestRepo(t)

	// make sure the branch does not exist
	cmd := exec.Command("git", "branch", "-D", "mrtest")
	cmd.Dir = repo
	cmd.CombinedOutput()

	first := exec.Command(labBinaryPath, "mr", "checkout", "1")
	first.Dir = repo
	b, err := first.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	second := exec.Command(labBinaryPath, "mr", "checkout", "1")
	second.Dir = repo
	log, err := second.CombinedOutput()
	eLog := string(log)
	if err == nil {
		t.Log(eLog)
		t.Fatal(err)
	}
	require.Contains(t, eLog, "branch mrtest already exists")
}

func Test_mrDoubleCheckoutForceRun(t *testing.T) {
	repo := copyTestRepo(t)

	// make sure the branch does not exist
	cmd := exec.Command("git", "branch", "-D", "mrtest")
	cmd.Dir = repo
	cmd.CombinedOutput()

	first := exec.Command(labBinaryPath, "mr", "checkout", "1")
	first.Dir = repo
	b, err := first.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	changeBranch := exec.Command("git", "checkout", "master")
	changeBranch.Dir = repo
	c, err := changeBranch.CombinedOutput()
	if err != nil {
		t.Log(string(c))
		t.Fatal(err)
	}

	second := exec.Command(labBinaryPath, "mr", "checkout", "1", "--force")
	second.Dir = repo
	log, err := second.CombinedOutput()
	eLog := string(log)
	if err != nil {
		t.Log(eLog)
		t.Fatal(err)
	}
	require.Contains(t, eLog, "Deleted branch mrtest")
}
