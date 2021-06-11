package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_mrCloseReopen(t *testing.T) {
	tests := []struct {
		desc     string
		opt      string
		expected string
	}{
		{
			desc:     "reopen-open",
			opt:      "reopen",
			expected: "mr not closed",
		},
		{
			desc:     "close-open",
			opt:      "close",
			expected: "Merge Request !740 closed",
		},
		{
			desc:     "close-closed",
			opt:      "close",
			expected: "mr already closed",
		},
		{
			desc:     "reopen-closed",
			opt:      "reopen",
			expected: "Merge Request !740 reopened",
		},
	}

	repo := copyTestRepo(t)
	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			cmd := exec.Command(labBinaryPath, "mr", test.opt, "740")
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
