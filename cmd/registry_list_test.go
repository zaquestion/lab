package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_registryList(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "registry", "list")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	projects := strings.Split(string(b), "\n")
	t.Log(projects)
	require.Equal(t, "lab-testing/www-gitlab-com", projects[0])
}
