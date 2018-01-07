package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_issueShow(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "issue", "1")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}

	require.Contains(t, string(b), `
#1 test issue for lab list
===================================

-----------------------------------
Project: zaquestion/test
Status: Open
Assignees: zaquestion, lab-testing
Author: lab-testing
Milestone: 1.0
Due Date: 2018-01-01 00:00:00 +0000 UTC
Time Stats: Estimated 1w, Spent 1d
Labels: bug
WebURL: https://gitlab.com/zaquestion/test/issues/1
`)
}
