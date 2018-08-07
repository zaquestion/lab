package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_projectList(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "project", "list", "-m")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	projects := strings.Split(string(b), "\n")
	t.Log(projects)
	require.Equal(t, "lab-testing/www-gitlab-com", projects[0])
}

func Test_projectList_many(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "project", "list", "-n", "101")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	projects := strings.Split(string(b), "\n")
	t.Log(projects[:len(projects)-3])
	require.Equal(t, "PASS", projects[len(projects)-3 : len(projects)-1][0])
	require.Contains(t, projects[len(projects)-2 : len(projects)][0], "of statements in ./...")
	require.Equal(t, 101, len(projects[:len(projects)-3]))
}
