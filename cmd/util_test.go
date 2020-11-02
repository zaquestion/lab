package cmd

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_textToMarkdown(t *testing.T) {
	basestring := "This string should have two spaces at the end."
	teststring := basestring + "\n"
	newteststring := textToMarkdown(teststring)
	assert.Equal(t, basestring+"  \n", newteststring)
}

func Test_getCurrentBranchMR(t *testing.T) {
	repo := copyTestRepo(t)

	// make sure the branch does not exist
	cmd := exec.Command("git", "branch", "-D", "mrtest")
	cmd.Dir = repo
	cmd.CombinedOutput()

	cmd = exec.Command(labBinaryPath, "mr", "checkout", "1")
	cmd.Dir = repo
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	curDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	err = os.Chdir(repo)
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	mrNum := getCurrentBranchMR("zaquestion/test")
	err = os.Chdir(curDir)
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	assert.Equal(t, 1, mrNum)
}
