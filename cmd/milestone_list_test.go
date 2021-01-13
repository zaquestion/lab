package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_milestoneList(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "milestone", "list")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	milestones := strings.Split(string(b), "\n")
	t.Log(milestones)
	require.Contains(t, milestones, "1.0")
}

func Test_milestoneListSearch(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "milestone", "list", "99")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	milestones := strings.Split(string(b), "\n")
	t.Log(milestones)
	require.NotContains(t, milestones, "1.0")
}

func Test_milestoneListState(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "milestone", "list", "--state", "closed")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	milestones := strings.Split(string(b), "\n")
	t.Log(milestones)
	require.NotContains(t, milestones, "1.0")
}
