package cmd

import (
	"os/exec"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/require"
)

func Test_issueShow(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "issue", "show", "1", "--comments")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}

	out := string(b)
	require.Contains(t, out, `
#1 test issue for lab list
===================================

-----------------------------------
Project: zaquestion/test
Status: Open
Assignees: zaquestion, lab-testing
Author: lab-testing
Milestone: 1.0
Due Date: 2018-01-01 00:00:00 +0000 UTC
Time Stats: Estimated 40h, Spent 8h
Labels: bug
Related MRs: 1
MRs that will close this Issue: 
WebURL: https://gitlab.com/zaquestion/test/-/issues/1
`)

	require.Contains(t, string(b), `commented at`)
}

func Test_issueShow_updated_comments(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "issue", "show", "8", "--comments")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}

	out := string(b)
	out = stripansi.Strip(out) // This is required because glamour adds a lot of ansi chars

	require.Contains(t, string(b), `updated comment at`)
}
