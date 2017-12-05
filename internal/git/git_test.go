package git

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	err := os.Chdir(os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata"))
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func TestGitDir(t *testing.T) {
	dir, err := GitDir()
	if err != nil {
		t.Fatal(err)
	}
	expectedDir := os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata/.git")
	require.Equal(t, expectedDir, dir)
}

func TestWorkingDir(t *testing.T) {
	dir, err := WorkingDir()
	if err != nil {
		t.Fatal(err)
	}
	expectedDir := os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata")
	require.Equal(t, expectedDir, dir)
}

func TestCommentChar(t *testing.T) {
	require.Equal(t, "#", CommentChar())
}

func TestLastCommitMessage(t *testing.T) {
	lcm, err := LastCommitMessage()
	if err != nil {
		t.Fatal(err)
	}
	expectedLCM := "Added additional commit for LastCommitMessage and meeting requirements for Log test (>1 commit)"
	require.Equal(t, expectedLCM, lcm)
}

func TestLog(t *testing.T) {
	log, err := Log("HEAD~1", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	expectedSHA := "cd64a7c"
	expectedAuthor := "Zaq? Wiedmann"
	expectedMessage := "Added additional commit for LastCommitMessage and meeting requirements for\n   Log test (>1 commit)\n\n"
	require.Contains(t, log, expectedSHA)
	require.Contains(t, log, expectedAuthor)
	require.Contains(t, log, expectedMessage)
}

func TestCurrentBranch(t *testing.T) {
	branch, err := CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	expectedBranch := "master"
	require.Equal(t, expectedBranch, branch)
}

func TestPathWithNameSpace(t *testing.T) {
	path, err := PathWithNameSpace("origin")
	if err != nil {
		t.Fatal(err)
	}
	expectedPath := "zaquestion/test"
	require.Equal(t, expectedPath, path)
}

func TestRepoName(t *testing.T) {
	repo, err := RepoName()
	if err != nil {
		t.Fatal(err)
	}
	expectedRepo := "test"
	require.Equal(t, expectedRepo, repo)
}

func TestIsRemote(t *testing.T) {
	res, err := IsRemote("origin")
	if err != nil {
		t.Fatal(err)
	}
	require.True(t, res)
}
