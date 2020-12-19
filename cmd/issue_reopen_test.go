package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_issueCloseReopen(t *testing.T) {
	tests := []struct {
		desc     string
		opt      string
		expected string
	}{
		{
			desc:     "reopen-open",
			opt:      "reopen",
			expected: "issue not closed",
		},
		{
			desc:     "close-open",
			opt:      "close",
			expected: "Issue #1 closed",
		},
		{
			desc:     "close-closed",
			opt:      "close",
			expected: "issue already closed",
		},
		{
			desc:     "reopen-closed",
			opt:      "reopen",
			expected: "Issue #1 reopened",
		},
	}

	repo := copyTestRepo(t)
	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			cmd := exec.Command(labBinaryPath, "issue", test.opt, "1")
			cmd.Dir = repo

			b, err := cmd.CombinedOutput()
			if err != nil {
				t.Log(string(b))
			}

			out := string(b)
			require.Contains(t, out, test.expected)
		})
	}
}
