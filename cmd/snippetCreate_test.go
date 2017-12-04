package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_snippetCreate(t *testing.T) {
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "snippet", "create", "lab-testing",
		"-m", "snippet title",
		"-m", "snippet description")
	cmd.Dir = repo

	rc, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}

	_, err = rc.Write([]byte("snippet contents"))
	if err != nil {
		t.Fatal(err)
	}
	err = rc.Close()
	if err != nil {
		t.Fatal(err)
	}

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	require.Contains(t, string(b), "https://gitlab.com/lab-testing/test/snippets/")
}
