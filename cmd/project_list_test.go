package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_projectList(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "project", "list", "-m")
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
	cmd := exec.Command(labBinaryPath, "project", "list", "-n", "101")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	projects := strings.Split(string(b), "\n")
	t.Log(projects)

	projects = truncAppOutput(projects)
	assert.Equal(t, 101, len(projects), "Expected 101 projects listed")
	assert.NotContains(t, projects, "PASS")
}

// truncAppOutput truncates the list of strings returned from the "lab" test
// app to remove the test-specific output. It use "PASS" as a marker for the end
// of the app output and the beginning of the test output.
func truncAppOutput(output []string) []string {
	for i, line := range output {
		if line == "PASS" {
			return output[:i]
		}
	}
	return output
}
