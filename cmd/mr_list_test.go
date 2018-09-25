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
	require.Contains(t, mrs, "#1 Test MR for lab list")
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
	require.Contains(t, mrs, "#3 for testings filtering with labels and lists")
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
	require.Contains(t, mrs, "#4 merged merge request")
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
	require.Contains(t, mrs, "#5 closed mr")

}

func Test_mrListFivePerPage(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "mr", "list", "-n", "5")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Contains(t, mrs, "#1 Test MR for lab list")
}

func Test_mrFilterByTargetBranch(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "mr", "list", "-t", "non-existing")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	require.Equal(t, "PASS", mrs[0])
}

func Test_mrListByTargetBranch(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "mr", "list", "-t", "master")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	require.Equal(t, "#107 WIP: Resolve \"issue title\"", mrs[0])
}
