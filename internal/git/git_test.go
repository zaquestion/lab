package git

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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
	expectedLCM := "(ci) jobs with interleaved sleeps and prints"
	require.Equal(t, expectedLCM, lcm)
}

func TestLog(t *testing.T) {
	log, err := Log("HEAD~1", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	expectedSHA := "09b519c"
	expectedAuthor := "Zaq? Wiedmann"
	expectedMessage := "(ci) jobs with interleaved sleeps and prints"
	assert.Contains(t, log, expectedSHA)
	assert.Contains(t, log, expectedAuthor)
	assert.Contains(t, log, expectedMessage)
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
	tests := []struct {
		desc        string
		remote      string
		expected    string
		expectedErr string
	}{
		{
			desc:        "ssh",
			remote:      "origin",
			expected:    "zaquestion/test",
			expectedErr: "",
		},
		{
			desc:        "http",
			remote:      "origin-http",
			expected:    "zaquestion/test",
			expectedErr: "",
		},
		{
			desc:        "https",
			remote:      "origin-https",
			expected:    "zaquestion/test",
			expectedErr: "",
		},
		{
			desc:        "https://token@gitlab.com/org/repo",
			remote:      "origin-https-token",
			expected:    "zaquestion/test",
			expectedErr: "",
		},
		{
			desc:        "git://",
			remote:      "origin-git",
			expected:    "zaquestion/test",
			expectedErr: "",
		},
		{
			desc:        "ssh://",
			remote:      "origin-ssh-alt",
			expected:    "zaquestion/test",
			expectedErr: "",
		},
		{
			desc:        "no .git suffix",
			remote:      "origin-no_dot_git",
			expected:    "zaquestion/test",
			expectedErr: "",
		},
		{
			desc:        "subdfolders-ssh",
			remote:      "origin-subfolder-ssh",
			expected:    "zaquestion/sub/folder/test",
			expectedErr: "",
		},
		{
			desc:        "subdfolders-git",
			remote:      "origin-subfolder-git",
			expected:    "zaquestion/sub/folder/test",
			expectedErr: "",
		},
		{
			desc:        "ssh-custom-port",
			remote:      "origin-custom-port",
			expected:    "zaquestion/test",
			expectedErr: "",
		},
		{
			desc:        "ssh-subfolder-custom-port",
			remote:      "origin-subfolder-custom-port",
			expected:    "zaquestion/sub/folder/test",
			expectedErr: "",
		},
		{
			desc:        "remote doesn't exist",
			remote:      "phoney",
			expected:    "",
			expectedErr: "the key `remote.phoney.url` is not found",
		},
		{
			desc:        "remote doesn't exist",
			remote:      "garbage",
			expected:    "",
			expectedErr: "cannot parse remote: garbage url: garbageurl",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			path, err := PathWithNameSpace(test.remote)
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.expected, path)
		})
	}
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

func TestInsideGitRepo(t *testing.T) {
	require.True(t, InsideGitRepo())
}
