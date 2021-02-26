package cmd

import (
	"os/exec"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/require"
)

func Test_issueSubscribeSetup(t *testing.T) {
	repo := copyTestRepo(t)
	orig := exec.Command(labBinaryPath, "issue", "show", "1")
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

func Test_issueSubscribe(t *testing.T) {
	repo := copyTestRepo(t)
	orig := exec.Command(labBinaryPath, "issue", "subscribe", "1")
	orig.Dir = repo

	b, err := orig.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}

	origOutput := string(b)
	origOutput = stripansi.Strip(origOutput)

	require.Contains(t, origOutput, `Subscribed to issue #1`)
}

func Test_issueUnsubscribe(t *testing.T) {
	repo := copyTestRepo(t)
	orig := exec.Command(labBinaryPath, "issue", "unsubscribe", "1")
	orig.Dir = repo

	b, err := orig.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}

	origOutput := string(b)
	origOutput = stripansi.Strip(origOutput)

	require.Contains(t, origOutput, `Unsubscribed from issue #1`)
}
