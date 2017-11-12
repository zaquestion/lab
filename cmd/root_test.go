package cmd

import (
	"log"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zaquestion/lab/internal/git"
)

func TestMain(m *testing.M) {
	wd, err := git.WorkingDir()
	if err != nil {
		log.Fatal(err)
	}
	os.Chdir(wd)
	err = exec.Command("go", "build", "-o", "lab_bin").Run()
	if err != nil {
		log.Fatal(err)
	}
	code := m.Run()
	os.Remove("lab_bin")
	os.Exit(code)

}

func TestRootCloneNoArg(t *testing.T) {
	cmd := exec.Command("./lab_bin", "clone")
	b, _ := cmd.CombinedOutput()
	require.Contains(t, string(b), "You must specify a repository to clone.")
}
