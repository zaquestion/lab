package cmd

import (
	"os/exec"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/require"
)

// https://gitlab.com/zaquestion/test/-/merge_requests/18 was opened for these
// tests

func Test_mrApproveSetup(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	orig := exec.Command(labBinaryPath, "mr", "show", "18")
	orig.Dir = repo

	b, err := orig.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}

	origOutput := string(b)
	origOutput = stripansi.Strip(origOutput)

	require.Contains(t, origOutput, `Approved By: None`)
}

func Test_mrApprove(t *testing.T) {
	repo := copyTestRepo(t)
	orig := exec.Command(labBinaryPath, "mr", "approve", "18")
	orig.Dir = repo

	b, err := orig.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}

	origOutput := string(b)
	origOutput = stripansi.Strip(origOutput)

	require.Contains(t, origOutput, `Merge Request #18 approved`)
}

func Test_mrUnapprove(t *testing.T) {
	repo := copyTestRepo(t)
	orig := exec.Command(labBinaryPath, "mr", "unapprove", "18")
	orig.Dir = repo

	b, err := orig.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}

	origOutput := string(b)
	origOutput = stripansi.Strip(origOutput)

	require.Contains(t, origOutput, `Merge Request #18 unapproved`)
}
