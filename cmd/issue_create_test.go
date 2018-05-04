package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_issueCreate(t *testing.T) {
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "issue", "create", "lab-testing",
		"-m", "issue title")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	require.Contains(t, string(b), "https://gitlab.com/lab-testing/test/issues/")
}

func Test_issueMsg(t *testing.T) {
	tests := []struct {
		Name          string
		Msgs          []string
		ExpectedTitle string
		ExpectedBody  string
	}{
		{
			Name:          "Using messages",
			Msgs:          []string{"issue title", "issue body", "issue body 2"},
			ExpectedTitle: "issue title",
			ExpectedBody:  "issue body\n\nissue body 2",
		},
		{
			Name:          "From Editor",
			Msgs:          nil,
			ExpectedTitle: "This is the default issue template for lab",
			ExpectedBody:  "",
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			title, body, err := issueMsg(test.Msgs)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, test.ExpectedTitle, title)
			assert.Equal(t, test.ExpectedBody, body)
		})
	}
}

func Test_issueText(t *testing.T) {
	t.Parallel()
	text, err := issueText()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, `

This is the default issue template for lab
# Write a message for this issue. The first block
# of text is the title and the rest is the description.`, text)

}
