package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_clone(t *testing.T) {
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "clone", "test")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	out := string(b)
	t.Log(out)

	assert.Contains(t, out, "Cloning into 'test'...")
	assert.Contains(t, out, " * [new branch]      master     -> upstream/master")
	assert.Contains(t, out, "new remote: upstream")
}
