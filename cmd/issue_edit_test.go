package cmd

import (
	"log"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// create an issue and return the issue number
func issueEditCmdTest_createIssue(dir string) string {
	cmd := exec.Command("../lab_bin", "issue", "create", "lab-testing",
		"-m", "issue title", "-l", "bug")
	cmd.Dir = dir

	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	s := strings.Split(string(b), "\n")
	s = strings.Split(s[0], "/")
	return s[len(s)-1]
}

func issueEditCmdTest_showIssue(dir string, issueNum string) string {
	cmd := exec.Command("../lab_bin", "issue", "show", "lab-testing", issueNum)
	cmd.Dir = dir

	b, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	return string(b)
}

func Test_issueEditCmd(t *testing.T) {
	repo := copyTestRepo(t)

	issueNum := issueEditCmdTest_createIssue(repo)

	// update the issue
	cmd := exec.Command("../lab_bin", "issue", "edit", "lab-testing", issueNum,
		"-m", "new title")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	// show the updated issue
	issueShowOuput := issueEditCmdTest_showIssue(repo, issueNum)

	// the output should show the updated title, not the old title
	require.Contains(t, issueShowOuput, "new title")
	require.NotContains(t, issueShowOuput, "issue title")
}

func Test_issueEditLabels(t *testing.T) {
	repo := copyTestRepo(t)

	issueNum := issueEditCmdTest_createIssue(repo)

	// update the issue
	cmd := exec.Command("../lab_bin", "issue", "edit", "lab-testing", issueNum,
		"-l", "critical", "-L", "bug")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	// show the updated issue
	issueShowOuput := issueEditCmdTest_showIssue(repo, issueNum)

	// the output should show the updated title, not the old title
	require.Contains(t, issueShowOuput, "critical")
	require.NotContains(t, issueShowOuput, "bug")
}

func Test_issueEditMsg(t *testing.T) {
	tests := []struct {
		Name          string
		CurrentTitle  string
		CurrentBody   string
		Msgs          []string
		ExpectedTitle string
		ExpectedBody  string
	}{
		{
			Name:          "Using messages",
			CurrentTitle:  "old title",
			CurrentBody:   "old body",
			Msgs:          []string{"new title", "new body 1", "new body 2"},
			ExpectedTitle: "new title",
			ExpectedBody:  "new body 1\n\nnew body 2",
		},
		{
			Name:          "Using a single message",
			CurrentTitle:  "old title",
			CurrentBody:   "old body",
			Msgs:          []string{"new title"},
			ExpectedTitle: "new title",
			ExpectedBody:  "old body",
		},
		{
			Name:          "From Editor",
			CurrentTitle:  "old title",
			CurrentBody:   "old body",
			Msgs:          nil,
			ExpectedTitle: "old title",
			ExpectedBody:  "old body",
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			title, body, err := issueEditMsg(test.CurrentTitle, test.CurrentBody, test.Msgs)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, test.ExpectedTitle, title)
			assert.Equal(t, test.ExpectedBody, body)
		})
	}
}

func Test_issueEditText(t *testing.T) {
	t.Parallel()
	text, err := issueEditText("old title", "old body")
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, `old title

old body

# Edit the title and/or description of this issue. The first
# block of text is the title and the rest is the description.`, text)

}

func Test_issueEditSame(t *testing.T) {
	t.Parallel()
	assert.True(t, same([]string{}, []string{}))
	assert.True(t, same([]string{"a"}, []string{"a"}))
	assert.True(t, same([]string{"a", "b"}, []string{"a", "b"}))
	assert.True(t, same([]string{"a", "b"}, []string{"b", "a"}))
	assert.True(t, same([]string{"b", "a"}, []string{"a", "b"}))

	assert.False(t, same([]string{"a"}, []string{}))
	assert.False(t, same([]string{"a"}, []string{"c"}))
	assert.False(t, same([]string{}, []string{"c"}))
	assert.False(t, same([]string{"a", "b"}, []string{"a", "c"}))
	assert.False(t, same([]string{"a", "b"}, []string{"a"}))
	assert.False(t, same([]string{"a", "b"}, []string{"c"}))
}

func Test_issueEditUnion(t *testing.T) {
	t.Parallel()
	s := union([]string{"a", "b"}, []string{"c"})
	assert.Equal(t, 3, len(s))
	assert.True(t, same(s, []string{"a", "b", "c"}))
}

func Test_issueEditDifference(t *testing.T) {
	t.Parallel()
	s := difference([]string{"a", "b"}, []string{"c"})
	assert.Equal(t, 2, len(s))
	assert.True(t, same(s, []string{"a", "b"}))

	s = difference([]string{"a", "b"}, []string{"a", "c"})
	assert.Equal(t, 1, len(s))
	assert.True(t, same(s, []string{"b"}))
}
