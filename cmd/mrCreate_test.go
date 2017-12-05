package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_mrCreate(t *testing.T) {
	repo := copyTestRepo(t)
	git := exec.Command("git", "checkout", "origin/mrtest")
	git.Dir = repo
	out, err := git.CombinedOutput()
	if err != nil {
		t.Log(string(out))
		t.Fatal(err)
	}
	git = exec.Command("git", "checkout", "-b", "mrtest")
	git.Dir = repo
	out, err = git.CombinedOutput()
	if err != nil {
		t.Log(string(out))
		t.Fatal(err)
	}

	cmd := exec.Command("../lab_bin", "mr", "create", "lab-testing",
		"-m", "mr title")
	cmd.Dir = repo

	b, _ := cmd.CombinedOutput()
	t.Log(string(b))
	// This message indicates that the GitLab API tried to create the MR,
	// its good enough to assert lab is working
	require.Contains(t, string(b), "409 {message: [Cannot Create: This merge request already exists: [\"mr title\"]]}")
}
