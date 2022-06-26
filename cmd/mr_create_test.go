package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// MR Create is tested in cmd/mr_test.go

func Test_mrText(t *testing.T) {
	text, err := mrText("origin", "mrtest", "origin", "master", false, false)
	if err != nil {
		t.Log(text)
		t.Fatal(err)
	}
	require.Contains(t, text, `

I am the default merge request template for lab
# Requesting a merge into origin:master from origin:mrtest (1 commits)
#
# Write a message for this merge request. The first block
# of text is the title and the rest is the description.
#
# Changes:
#
# 54fd49a`)

}

func Test_mrText_CoverLetter(t *testing.T) {
	coverLetter, err := mrText("origin", "mrtest", "origin", "master", true, false)
	if err != nil {
		t.Log(coverLetter)
		t.Fatal(err)
	}
	require.Contains(t, coverLetter, `

I am the default merge request template for lab
# Requesting a merge into origin:master from origin:mrtest (1 commits)
#
# Write a message for this merge request. The first block
# of text is the title and the rest is the description.
#
# Changes:
#

54fd49a`)

}
