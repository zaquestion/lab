package cmd

import (
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

func cleanupFork(t *testing.T, project string) {
	// Failing to find the project will fail the test and is a legit
	// failure case since its the only thing asserting the project exists
	// (was forked)
	p, err := lab.FindProject(project)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to find project "+project+" for cleanup"))
	}
	err = lab.ProjectDelete(p.ID)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to delete project "+project+" during cleanup"))
	}
}

func Test_fork(t *testing.T) {
	tests := []struct {
		desc      string
		args      []string
		name      string
		path      string
		namespace string
	}{
		{
			desc:      "do_fork",
			args:      []string{},
			name:      "fork_test",
			path:      "",
			namespace: "",
		},
		{
			desc:      "do_fork_name",
			args:      []string{"-n", "fork_test_name"},
			name:      "fork_test_name",
			path:      "",
			namespace: "",
		},
		{
			desc:      "do_fork_path",
			args:      []string{"-p", "fork_test_path"},
			name:      "fork_test",
			path:      "fork_test_path",
			namespace: "",
		},
		{
			desc:      "do_fork_name_path",
			args:      []string{"-n", "fork_test_name_1", "-p", "fork_test_path_1"},
			name:      "fork_test_name_1",
			path:      "fork_test_path_1",
			namespace: "",
		},
		{
			desc:      "do_fork_namespace",
			args:      []string{"-g", "lab-testing-test-group"},
			name:      "fork_test",
			path:      "",
			namespace: "lab-testing-test-group",
		},
		{
			desc:      "do_fork_namespace_name",
			args:      []string{"-g", "lab-testing-test-group", "-n", "fork_test_name"},
			name:      "fork_test_name",
			path:      "",
			namespace: "lab-testing-test-group",
		},
		{
			desc:      "do_fork_namespace_path",
			args:      []string{"-g", "lab-testing-test-group", "-p", "fork_test_path"},
			name:      "fork_test",
			path:      "fork_test_path",
			namespace: "lab-testing-test-group",
		},
		{
			desc:      "do_fork_namespace_name_path",
			args:      []string{"-g", "lab-testing-test-group", "-n", "fork_test_name_1", "-p", "fork_test_path_1"},
			name:      "fork_test_name_1",
			path:      "fork_test_path_1",
			namespace: "lab-testing-test-group",
		},
	}

	t.Parallel()

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			// remove the .git/config so no remotes exist
			repo := copyTestRepo(t)
			os.Remove(path.Join(repo, ".git/config"))

			cmd := exec.Command("git", "remote", "add", "origin", "git@gitlab.com:zaquestion/fork_test")
			cmd.Dir = repo
			err := cmd.Run()
			if err != nil {
				t.Fatal(err)
			}

			namespace := "lab-testing"
			if test.namespace != "" {
				namespace = test.namespace
			}
			name := test.name
			if test.path != "" {
				name = test.path
			}
			project := namespace + "/" + name

			args := []string{"fork"}
			if len(test.args) > 0 {
				args = append(args, test.args...)
			}
			cmd = exec.Command(labBinaryPath, args...)
			cmd.Dir = repo
			b, err := cmd.CombinedOutput()
			out := string(b)
			if err != nil {
				t.Log(out)
				cleanupFork(t, project)
				t.Fatal(err)
			}

			require.Contains(t, out, "From gitlab.com:"+project)
			require.Contains(t, out, "new remote: "+namespace)

			cmd = exec.Command("git", "remote", "-v")
			cmd.Dir = repo

			b, err = cmd.CombinedOutput()
			if err != nil {
				t.Fatal(err)
			}
			require.Contains(t, string(b), namespace)
			time.Sleep(2 * time.Second)
			cleanupFork(t, project)
		})
	}
}

func Test_forkWait(t *testing.T) {
	repo := copyTestRepo(t)
	os.Remove(path.Join(repo, ".git/config"))

	cmd := exec.Command("git", "remote", "add", "origin", "git@gitlab.com:zaquestion/fork_test")
	cmd.Dir = repo
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	// The default behavior is to "wait" for the fork and it's already being
	// tested in the Test_fork() test. Here we only test the --no-wait
	// option, which we can't effectively test because we don't have a big
	// enough repo.
	t.Run("fork_nowait", func(t *testing.T) {
		cmd = exec.Command(labBinaryPath, []string{"fork", "--no-wait"}...)
		cmd.Dir = repo
		b, err := cmd.CombinedOutput()
		out := string(b)
		if err != nil {
			t.Log(out)
			t.Fatal(err)
		}
		require.Contains(t, out, "From gitlab.com:lab-testing/fork_test")
		require.Contains(t, out, "new remote: lab-testing")
		cleanupFork(t, "fork_test")
	})
}

func Test_determineForkRemote(t *testing.T) {
	tests := []struct {
		desc     string
		custom   string
		project  string
		expected string
	}{
		{"project is forked from repo", "", "zaquestion", "lab-testing"},
		{"project is user", "", "lab-testing", "upstream"},
		{"project is user", "custom-test", "lab-testing", "custom-test"},
	}

	for _, test := range tests {
		test := test
		remoteName = test.custom
		t.Run(test.desc, func(t *testing.T) {
			require.Equal(t, test.expected, determineForkRemote(test.project))
		})
	}
}
