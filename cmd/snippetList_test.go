package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_snippetList(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "snippet", "list", "lab-testing")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	snips := strings.Split(string(b), "\n")
	t.Log(snips)
	require.Regexp(t, `#\d+ snippet title`, snips[0])
}

func Test_snippetList_Global(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "snippet", "list", "-g")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	snips := strings.Split(string(b), "\n")
	t.Log(snips)
	var ok bool
	for i := 0; i < 3; i++ {
		// Tests are running in a parallel and project snippets are
		// included in the personal snippet lists. Still we want to
		// make sure personal snippets are getting created. One should
		// be guaranteed to be in the top 3, since a given test run
		// only creates 2 personal and 1 project snippet
		if ok = assert.Regexp(t, `#\d+ personal snippet title`, snips[i]); ok {
			break
		}
	}
	require.True(t, ok)
}
