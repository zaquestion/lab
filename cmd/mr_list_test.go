package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_mrList(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "mr", "list")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Equal(t, "#1 Test MR for lab list", mrs[0])
}

func Test_mrListFlagLabel(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "mr", "list", "-l", "confirmed")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Equal(t, "#3 Test filtering by label", mrs[0])
}

func Test_mrListStateMerged(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "mr", "list", "-s", "merged")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Equal(t, "#4 merged merge request", mrs[0])
}

func Test_mrListStateClosed(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "mr", "list", "-s", "closed")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Equal(t, "#5 closed mr", mrs[0])
}

func Test_mrListFivePerPage(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "mr", "list", "-n", 5)
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Equal(t, "#6 with per page preset", mrs[0])
}
