package cmd

import (
	"os/exec"
	"strings"
	"testing"

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
	cmd := exec.Command("../lab_bin", "snippet", "-l", "-g")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	snips := strings.Split(string(b), "\n")
	t.Log(snips)
	require.Regexp(t, `#\d+ personal snippet title`, snips[:3])
}
