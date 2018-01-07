package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_fork(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	// remove the remote so we can test adding it back
	// NOTE: we aren't actually going to test that forks are created on
	// GitLab, just that lab behaves correctly when a fork exists.
	cmd := exec.Command("git", "remote", "remove", "lab-testing")
	cmd.Dir = repo
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	cmd = exec.Command("../lab_bin", "fork")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	out := string(b)

	require.Contains(t, out, "From gitlab.com:lab-testing/test")
	require.Contains(t, out, "new remote: lab-testing")

	cmd = exec.Command("git", "remote", "-v")
	cmd.Dir = repo

	b, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	require.Contains(t, string(b), "lab-testing")
}
