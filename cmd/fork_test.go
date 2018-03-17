package cmd

import (
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

func Test_fork(t *testing.T) {
	t.Parallel()

	repo := copyTestRepo(t)

	// remove the .git/config so no remotes exist
	os.Remove(path.Join(repo, ".git/config"))

	cmd := exec.Command("git", "remote", "add", "origin",
		"git@gitlab.com:zaquestion/fork_test")
	cmd.Dir = repo
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("do_fork", func(t *testing.T) {
		cmd = exec.Command("../lab_bin", "fork")
		cmd.Dir = repo
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}

		out := string(b)

		require.Contains(t, out, "From gitlab.com:lab-testing/fork_test")
		require.Contains(t, out, "new remote: lab-testing")

		cmd = exec.Command("git", "remote", "-v")
		cmd.Dir = repo

		b, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}
		require.Contains(t, string(b), "lab-testing")
	})

	// Failing to find the project will fail the test and is a legit
	// failure case since its the only thing asserting the project exists
	// (was forked)
	p, err := lab.FindProject("fork_test")
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to find project for cleanup"))
	}
	err = lab.ProjectDelete(p.ID)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to delete project during cleanup"))
	}
}

func Test_determineForkRemote(t *testing.T) {
	tests := []struct {
		desc     string
		project  string
		expected string
	}{
		{"project is forked from repo", "zaquestion", "lab-testing"},
		{"project is user", "lab-testing", "upstream"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			require.Equal(t, test.expected, determineForkRemote(test.project))
		})
	}
}
