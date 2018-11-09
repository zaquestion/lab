package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_issueList(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "issue", "list")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	issues := strings.Split(string(b), "\n")
	t.Log(issues)
	require.Contains(t, issues, "#1 test issue for lab list")
}

func Test_issueListFlagLabel(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "issue", "list", "-l", "enhancement")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	issues := strings.Split(string(b), "\n")
	t.Log(issues)
	require.Contains(t, issues, "#3 test filter labels 1")
}

func Test_issueListStateClosed(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "issue", "list", "-s", "closed")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	issues := strings.Split(string(b), "\n")
	t.Log(issues)
	require.Contains(t, issues, "#4 test closed issue")
}

func Test_issueListSearch(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "issue", "list", "filter labels")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	issues := strings.Split(string(b), "\n")
	t.Log(issues)
	require.Contains(t, issues, "#3 test filter labels 1")
	require.NotContains(t, issues, "#1 test issue for lab list")
}
