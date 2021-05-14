package git

import (
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	repo := copyTestRepo()
	if err := os.Chdir(repo); err != nil {
		log.Fatal(err)
	}

	code := m.Run()

	if err := os.Chdir("../"); err != nil {
		log.Fatalf("Error chdir to ../: %s", err)
	}
	if err := os.RemoveAll(repo); err != nil {
		log.Fatalf("Error removing %s: %s", repo, err)
	}
	os.Exit(code)
}

func TestGitDir(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, filepath.Clean(wd+"/.git"), filepath.Clean(dir))
}

func TestWorkingDir(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := WorkingDir()
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, filepath.Clean(wd), filepath.Clean(dir))
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
	count := NumberCommits("HEAD~1", "HEAD")
	expectedSHA := "09b519c"
	expectedAuthor := "Zaq? Wiedmann"
	expectedMessage := "(ci) jobs with interleaved sleeps and prints"
	assert.Contains(t, log, expectedSHA)
	assert.Contains(t, log, expectedAuthor)
	assert.Contains(t, log, expectedMessage)
	assert.Equal(t, 1, count)
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
			desc:        "pushurl",
			remote:      "origin-pushurl",
			expected:    "zaquestion/test",
			expectedErr: "",
		},
		{
			desc:        "empty-pushurl",
			remote:      "origin-empty-pushurl",
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

func TestRemotes(t *testing.T) {
	res, err := Remotes()
	if err != nil {
		t.Fatal(err)
	}
	require.Contains(t, res, "origin", "remotes should contain 'origin' [%v]", res)
}

func TestRemoteBranches(t *testing.T) {
	res, err := RemoteBranches("origin")
	if err != nil {
		t.Fatal(err)
	}
	require.Contains(t, res, "master", "remote branches should contain 'master' [%v]", res)
}

// Make sure the git command follow the expected behavior.
func TestGetLocalRemotesFromFile(t *testing.T) {
	repo := copyTestRepo()
	gitconfig := repo + "/.git/config"

	// First get the remotes directly from file
	cmd := exec.Command("grep", "^\\[remote.*", gitconfig)
	cmd.Dir = repo
	grep, err := cmd.Output()
	if err != nil {
		t.Log(string(grep))
		t.Error(err)
	}
	// And get the remote names
	re := regexp.MustCompile(`\[remote\s+"(.*)".*`)
	reMatches := re.FindAllStringSubmatch(string(grep), -1)

	// Second, get result from local function
	res, err := GetLocalRemotesFromFile()
	if err != nil {
		t.Log(string(res))
		t.Error(err)
	}
	remotesList := strings.Split(string(res), "\n")

	// Create an array (the ordering matters) to place all unique
	// remote names
	var remoteNames []string
	for _, remote := range remotesList {
		if len(remote) == 0 {
			continue
		}

		name := strings.Split(remote, ".")[1]
		// Check if name is unique
		var found bool
		for _, placedName := range remoteNames {
			if name == placedName {
				found = true
				break
			}
		}
		if found {
			continue
		}
		remoteNames = append(remoteNames, name)
	}

	// Check if remote exists and is in order
	for i, match := range reMatches {
		require.Equal(t, match[1], remoteNames[i])
	}
}

// copyTestRepo creates a copy of the testdata directory (contains a Git repo) in
// the project root with a random dir name. It returns the absolute path of the
// new testdata dir.
// Note: testdata-* must be in the .gitignore or the copies will create write
// errors as Git attempts to add the Git repo to the the project repo's index.
func copyTestRepo() string {
	dst, err := filepath.Abs(os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata-" + strconv.Itoa(int(rand.Uint64()))))
	if err != nil {
		log.Fatal(err)
	}
	src, err := filepath.Abs(os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata"))
	if err != nil {
		log.Fatal(err)
	}
	if err := copy.Copy(src, dst); err != nil {
		log.Fatal(err)
	}
	// Move the test.git dir into the expected path at .git
	if err := copy.Copy(dst+"/test.git", dst+"/.git"); err != nil {
		log.Fatal(err)
	}
	return dst
}
