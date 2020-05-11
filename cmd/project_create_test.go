package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

func Test_projectCreateCmd(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	expectedPath := filepath.Base(repo)

	// remove the .git/config so no remotes exist
	os.Remove(filepath.Join(repo, ".git/config"))

	t.Run("create", func(t *testing.T) {
		cmd := exec.Command(labBinaryPath, "project", "create", "-p")
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}

		require.Contains(t, string(b), "https://gitlab.com/lab-testing/"+expectedPath+"\n")

		gitCmd := git.New("remote", "get-url", "origin")
		gitCmd.Dir = repo
		gitCmd.Stdout = nil
		gitCmd.Stderr = nil
		remote, err := gitCmd.CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, "git@gitlab.com:lab-testing/"+expectedPath+".git\n", string(remote))
	})
	p, err := lab.FindProject(expectedPath)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to find project for cleanup"))
	}
	err = lab.ProjectDelete(p.ID)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to delete project during cleanup"))
	}
}

func Test_determinePath(t *testing.T) {
	t.Parallel()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		desc     string
		args     []string
		expected string
	}{
		{"arguemnt", []string{"new_project"}, "new_project"},
		{"git working dir", []string{}, filepath.Base(wd)},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.expected, determinePath(test.args, ""))
		})
	}
}
