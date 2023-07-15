package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_mrListAssignedTo(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "--assignee=zaquestion")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Contains(t, mrs, "!1 Test MR for lab list")
	require.NotContains(t, mrs, "!329 MR for assign and review commands")
	require.NotContains(t, mrs, "!3 for testings filtering with labels and lists")
}

func Test_mrListAssignedToAny(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "--assignee=any")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Contains(t, mrs, "!1 Test MR for lab list")
	require.Contains(t, mrs, "!329 MR for assign and review commands")
	require.NotContains(t, mrs, "!3 for testings filtering with labels and lists")
}

func Test_mrListAssignedToNone(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "--assignee=none")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.NotContains(t, mrs, "!1 Test MR for lab list")
	require.NotContains(t, mrs, "!329 MR for assign and review commands")
	require.Contains(t, mrs, "!3 for testings filtering with labels and lists")
}

func Test_mrList(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Contains(t, mrs, "!1 Test MR for lab list")
}

func Test_mrListFlagLabel(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "-l", "confirmed")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Contains(t, mrs, "!3 for testings filtering with labels and lists")
}

func Test_mrListStateMerged(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "-s", "merged")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Contains(t, mrs, "!4 merged merge request")
}

func Test_mrListStateClosed(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "-n", "1", "-s", "closed")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	require.Regexp(t, `!\d+`, string(b))
}

func Test_mrListFivePerPage(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "-n", "5")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	mrs := getAppOutput(b)
	require.Len(t, mrs, 5)
}

func Test_mrFilterByTargetBranch(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "-t", "non-existing")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := getAppOutput(b)
	assert.Empty(t, mrs, "Expected to find no MRs for non-existent branch")
}

var (
	latestCreatedTestMR = "!1447 Test for mr approval rules"
	latestUpdatedTestMR = "!329 MR for assign and review commands"
)

func Test_mrListByTargetBranch(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "-t", "master")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	require.Equal(t, latestUpdatedTestMR, mrs[0])
}

// updated,asc
// !1
func Test_mrListUpdatedAscending(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "--number=1", "--order=updated_at", "--sort=asc")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Contains(t, mrs, "!3 for testings filtering with labels and lists")
}

// updatead,desc
// !18
func Test_mrListUpdatedDescending(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "--number=1", "--order=updated_at", "--sort=desc")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Equal(t, latestUpdatedTestMR, mrs[0])
}

// created,asc
// !1
func Test_mrListCreatedAscending(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "--number=1", "--order=created_at", "--sort=asc")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Contains(t, mrs, "!1 Test MR for lab list")
}

// created,desc
// !18
func Test_mrListCreatedDescending(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "--number=1", "--order=created_at", "--sort=desc")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Equal(t, latestCreatedTestMR, mrs[0])
}

func Test_mrListSearch(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "emoji")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Contains(t, mrs, "!6 test award emoji")
}

func Test_mrListFlagMilestone(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "--milestone", "1.0")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Contains(t, mrs, "!1 Test MR for lab list")
}

func Test_mrListFlagMilestoneAny(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "--milestone", "any")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Contains(t, mrs, "!1 Test MR for lab list")
}

func Test_mrListFlagMilestoneNone(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "--milestone", "none")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.NotContains(t, mrs, "!1 Test MR for lab list")
}

func Test_mrListApprover(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	// cmd := exec.Command(labBinaryPath, "mr", "approve", "1")
	cmd := exec.Command(labBinaryPath, "mr", "list", "--approver=lab-testing")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Contains(t, mrs, "!1 Test MR for lab list")
	require.NotContains(t, mrs, "!3 for testings filtering with labels and lists")
}

func Test_mrListApproverAny(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "--approver=any")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Contains(t, mrs, "!1 Test MR for lab list")
	require.NotContains(t, mrs, "!3 for testings filtering with labels and lists")
}

func Test_mrListApproverNone(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "--approver=none")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.NotContains(t, mrs, "!1 Test MR for lab list")
	require.Contains(t, mrs, "!3 for testings filtering with labels and lists")
}

func Test_mrListReviewer(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "--reviewer=lab-testing")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.NotContains(t, mrs, "!1 Test MR for lab list")
	require.Contains(t, mrs, "!329 MR for assign and review commands")
}

func Test_mrListReviewerAny(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "--reviewer=any")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.NotContains(t, mrs, "!1 Test MR for lab list")
	require.Contains(t, mrs, "!329 MR for assign and review commands")
}

func Test_mrListReviewerNone(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "list", "--reviewer=none")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	mrs := strings.Split(string(b), "\n")
	t.Log(mrs)
	require.Contains(t, mrs, "!1 Test MR for lab list")
	require.NotContains(t, mrs, "!329 MR for assign and review commands")
}
