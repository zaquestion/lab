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
	out = stripansi.Strip(out)
	require.Contains(t, out, `
#1 Test MR for lab list
===================================

  This MR is to remain open for testing the  lab mr list  functionality         


-----------------------------------
Project: zaquestion/test
Branches: mrtest->master
Status: Open
Assignee: zaquestion
Author: zaquestion
Milestone: 1.0
Labels: documentation
WebURL: https://gitlab.com/zaquestion/test/-/merge_requests/1
`)

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
