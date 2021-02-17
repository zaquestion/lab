package cmd

import (
	"os/exec"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/require"
)

// https://gitlab.com/zaquestion/test/-/merge_requests/18 was opened for these
// tests

func Test_mrSubscribeSetup(t *testing.T) {
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

	require.Contains(t, origOutput, `Subscribed: No`)
}

func Test_mrSubscribe(t *testing.T) {
	repo := copyTestRepo(t)
	orig := exec.Command(labBinaryPath, "mr", "subscribe", "18")
	orig.Dir = repo

	b, err := orig.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}

	origOutput := string(b)
	origOutput = stripansi.Strip(origOutput)

	require.Contains(t, origOutput, `Subscribed to merge request !18`)
}

func Test_mrUnsubscribe(t *testing.T) {
	repo := copyTestRepo(t)
	orig := exec.Command(labBinaryPath, "mr", "unsubscribe", "18")
	orig.Dir = repo

	b, err := orig.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}

	origOutput := string(b)
	origOutput = stripansi.Strip(origOutput)

	require.Contains(t, origOutput, `Unsubscribed from merge request !18`)
}
