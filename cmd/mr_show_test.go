package cmd

import (
	"os/exec"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/require"
)

func Test_mrShow(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)

	// a comment has been added to
	// https://gitlab.com/zaquestion/test/-/merge_requests/1 for this test
	cmd := exec.Command(labBinaryPath, "mr", "show", "1", "--comments")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}

	out := string(b)
	require.Contains(t, out, `
!1 Test MR for lab list
===================================
This MR is to remain open for testing the `+"`lab mr list`"+` functionality
-----------------------------------
Project: zaquestion/test
Branches: mrtest->master
Status: Open
Assignee: zaquestion
Author: zaquestion
Approved By: None
Milestone: 1.0
Labels: documentation
Issues Closed by this MR: 
Subscribed: Yes
WebURL: https://gitlab.com/zaquestion/test/-/merge_requests/1`)
	require.Contains(t, string(b), `commented at`)
	require.Contains(t, string(b), `updated comment at`)
}

func Test_mrShow_patch(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "show", "origin", "1", "--patch")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}

	out := string(b)
	out = stripansi.Strip(out)
	// The index line below has been stripped as it is dependent on
	// the git version and pretty defaults.
	require.Contains(t, out, `commit 54fd49a2ac60aeeef5ddc75efecd49f85f7ba9b0
Author: Zaq? Wiedmann <zaquestion@gmail.com>
Date:   Tue Sep 19 03:55:16 2017 +0000

    Test file for MR test

diff --git a/mrtest b/mrtest
new file mode 100644
`)
}

func Test_mrShow_diffs(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	coCmd := exec.Command(labBinaryPath, "mr", "checkout", "17")
	coCmd.Dir = repo
	coCmdOutput, err := coCmd.CombinedOutput()
	if err != nil {
		t.Log(string(coCmdOutput))
		t.Error(err)
	}

	cmd := exec.Command(labBinaryPath, "mr", "show", "origin", "17", "--comments")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}

	out := string(b)
	out = stripansi.Strip(out)

	coCmd = exec.Command("git", "checkout", "master")
	coCmd.Dir = repo
	coCmdOutput, err = coCmd.CombinedOutput()
	if err != nil {
		t.Log(string(coCmdOutput))
		t.Error(err)
	}

	require.Contains(t, out, `
File:test
  4   4  
  5   5  
  6   6  line 6 This is a test file with some text in it.
  7   7  
  8     -line 8 This is the second test line in the file.
      8 +line 8 This is an edit of line 8.


    This is a comment on the deleted line 8.
`)

	require.Contains(t, out, `
File:test
      6 +line 6 This is a test file with some text in it.
      7 +
      8 +line 8 This is an edit of line 8.
      9 +
     10 +line 10 This is the third line in the file.


    This is a comment on line 10.
`)
}
