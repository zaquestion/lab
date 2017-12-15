package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_mrCreate(t *testing.T) {
	repo := copyTestRepo(t)

	git := exec.Command("git", "checkout", "mrtest")
	git.Dir = repo
	out, err := git.CombinedOutput()
	if err != nil {
		t.Log(string(out))
		t.Fatal(err)
	}

	cmd := exec.Command("../lab_bin", "mr", "create", "lab-testing",
		"-m", "mr title")
	cmd.Dir = repo

	b, _ := cmd.CombinedOutput()
	t.Log(string(b))
	// This message indicates that the GitLab API tried to create the MR,
	// its good enough to assert lab is working
	require.Contains(t, string(b), "409 {message: [Cannot Create: This merge request already exists: [\"mr title\"]]}")
}

func Test_mrText(t *testing.T) {
	text, err := mrText("master", "mrtest", "lab-testing", "origin")
	if err != nil {
		t.Log(text)
		t.Fatal(err)
	}
	require.Contains(t, text, `Added additional commit for LastCommitMessage and meeting requirements for Log test (>1 commit)

I am the mr tmpl
# Requesting a merge into origin:master from lab-testing:mrtest
#
# Write a message for this merge request. The first block
# of text is the title and the rest is the description.
#
# Changes:
#
# 54fd49a (Zaq? Wiedmann`)

}
