package cmd

import (
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zaquestion/lab/internal/git"
)

func TestMain(m *testing.M) {
	wd, err := git.WorkingDir()
	if err != nil {
		log.Fatal(err)
	}
	os.Chdir(wd)
	err = exec.Command("go", "test", "-c", "-coverpkg", "./...", "-covermode", "count", "-o", "lab_bin").Run()
	if err != nil {
		log.Fatal(err)
	}
	rand.Seed(time.Now().UnixNano())
	os.Chdir(path.Join(wd, "testdata"))
	code := m.Run()
	os.Chdir(wd)
	os.Remove("lab_bin")
	testdirs, err := filepath.Glob("testdata-*")
	if err != nil {
		log.Fatal(err)
	}
	for _, dir := range testdirs {
		err := os.RemoveAll(dir)
		if err != nil {
			log.Fatal(err)
		}
	}

	os.Exit(code)
}

func copyTestRepo(t *testing.T) string {
	dir := "../testdata-" + strconv.Itoa(int(rand.Uint64()))
	t.Log(dir)
	err := exec.Command("cp", "-r", "../testdata", dir).Run()
	if err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir = path.Join(wd, dir)
	return dir
}

func TestRootCloneNoArg(t *testing.T) {
	cmd := exec.Command("../lab_bin", "clone")
	b, _ := cmd.CombinedOutput()
	require.Contains(t, string(b), "You must specify a repository to clone.")
}
