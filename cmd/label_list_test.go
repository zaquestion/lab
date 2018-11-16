package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_labelList(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "label", "list")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	labels := strings.Split(string(b), "\n")
	t.Log(labels)
	require.Contains(t, labels, "bug")
	require.Contains(t, labels, "confirmed")
	require.Contains(t, labels, "critical")
}

func Test_labelListSearch(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "label", "list", "bug")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	labels := strings.Split(string(b), "\n")
	t.Log(labels)
	require.Contains(t, labels, "bug")
	require.NotContains(t, labels, "confirmed")
}

func Test_labelListSearchCaseInsensitive(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "label", "list", "BUG")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	labels := strings.Split(string(b), "\n")
	t.Log(labels)
	require.Contains(t, labels, "bug")
	require.NotContains(t, labels, "confirmed")
}
