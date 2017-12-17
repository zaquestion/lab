package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_mrShow(t *testing.T) {
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "mr", "4")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}

	require.Contains(t, string(b), `
#4 merged merge request
===================================

-----------------------------------
Project: zaquestion/test
Branches: merged->master
Status: Merged
Work in Progress: false
Assignee: None
Author: zaquestion
Milestone: None
Labels: None
WebURL: https://gitlab.com/zaquestion/test/merge_requests/4
`)
}
