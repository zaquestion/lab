package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_issueShow(t *testing.T) {
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
Assignee: zaquestion
Author: lab-testing
Milestone: None
Due Date: None
Time Stats: None
Labels: bug
WebURL: https://gitlab.com/zaquestion/test/issues/1
`)
}
