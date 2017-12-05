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
	title, body, err := issueMsg([]string{"issue title", "issue body", "issue body 2"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "issue title", title)
	assert.Equal(t, "issue body\n\nissue body 2", body)

}

func Test_issueText(t *testing.T) {
	text, err := issueText()
	if err != nil {
		t.Fatal(err)
	}
	// Normally we we expect the issue template to prefix this. However
	// since `issueText()` is being called from the `cmd` directory the
	// underlying LoadGitLabTmpl call doesn't find a template.
	// This is fine since we have other tests to test loading the template
	assert.Equal(t, `

# Write a message for this issue. The first block
# of text is the title and the rest is the description.`, text)

}
