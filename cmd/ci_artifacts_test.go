package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ciArtifacts(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = repo
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	cmd = exec.Command(labBinaryPath, "ci", "artifacts", "origin", "master:build3:artifacts")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	out := string(b)
	assert.Contains(t, out, "Downloaded artifacts.zip")
}

func Test_ciArtifactsPath(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = repo
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	cmd = exec.Command(labBinaryPath, "ci", "artifacts", "-p", "artifact", "origin", "master:build3:artifacts")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	out := string(b)
	assert.Contains(t, out, "Downloaded artifact")
}
