package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// MR Create is tested in cmd/mr_test.go

func Test_mrText(t *testing.T) {
	t.Parallel()
	text, err := mrText("master", "mrtest", "lab-testing", "origin")
	if err != nil {
		t.Log(text)
		t.Fatal(err)
	}
	require.Contains(t, text, `(ci) jobs with interleaved sleeps and prints

I am the default merge request template for lab
# Requesting a merge into origin:master from lab-testing:mrtest
#
# Write a message for this merge request. The first block
# of text is the title and the rest is the description.
#
# Changes:
#
# 54fd49a (Zaq? Wiedmann`)

}
