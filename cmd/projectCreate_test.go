package cmd

import (
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

func Test_projectCreateCmd(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	parts := strings.Split(repo, "/")
	expectedPath := parts[len(parts)-1]

	// remove the .git/config so no remotes exist
	os.Remove(path.Join(repo, ".git/config"))

	t.Run("create", func(t *testing.T) {
		cmd := exec.Command("../lab_bin", "project", "create")
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
	tests := []struct {
		desc     string
		args     []string
		expected string
	}{
		{"arguemnt", []string{"new_project"}, "new_project"},
		// All cmd package tests run in the lab/testdata directory
		{"git working dir", []string{}, "testdata"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.expected, determinePath(test.args, ""))
		})
	}
}
