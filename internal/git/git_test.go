package git

import (
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/go-git.v4"
)

func TestMain(m *testing.M) {
	err := os.Chdir(os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata"))
	if err != nil {
		log.Fatal(err)
	}
	repo, err = git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func TestGitDir(t *testing.T) {
	wd, _ := os.Getwd()
	t.Log(wd)
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
	assert.Equal(t, strings.Count(log, "\n"), 3, "Expected only 1 revision to be shown")
	assert.Regexp(t, `09b519c \(Zaq\? Wiedmann, \d+ (year|month|day)s? ago\)
   \(ci\) jobs with interleaved sleeps and prints`, log)
}

func Test_wrap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		desc     string
		input    string
		expected string
	}{
		{
			desc:     "long line",
			input:    "foo_12345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890",
			expected: "foo_12345678901234567890123456789012345678901234567890123456789012345678901234\n567890123456789012345678901234567890",
		},
		{
			desc:     "short line",
			input:    "123456789",
			expected: "123456789",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.expected, wrap(test.input))
		})
	}
}

func Test_indent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		desc     string
		input    string
		expected string
	}{
		{
			desc:     "many lines",
			input:    "foo\nbar\nbiz\nbaz",
			expected: "   foo\n   bar\n   biz\n   baz",
		},
		{
			desc:     "single line",
			input:    "foo",
			expected: "   foo",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.expected, indent(test.input))
		})
	}
}

func Test_ago(t *testing.T) {
	t.Parallel()
	tests := []struct {
		desc     string
		dur      time.Duration
		expected string
	}{
		{
			desc:     "years ago",
			dur:      time.Now().Sub(time.Now().Add(-2 * time.Hour * 24 * 31 * 12)),
			expected: "2 years ago",
		},
		{
			desc:     "year ago",
			dur:      time.Now().Sub(time.Now().Add(-1 * time.Hour * 24 * 31 * 12)),
			expected: "1 year ago",
		},
		{
			desc:     "months ago",
			dur:      time.Now().Sub(time.Now().Add(-2 * time.Hour * 24 * 31)),
			expected: "2 months ago",
		},
		{
			desc:     "month ago",
			dur:      time.Now().Sub(time.Now().Add(-1 * time.Hour * 24 * 31)),
			expected: "1 month ago",
		},
		{
			desc:     "days ago",
			dur:      time.Now().Sub(time.Now().Add(-2 * time.Hour * 24)),
			expected: "2 days ago",
		},
		{
			desc:     "day ago",
			dur:      time.Now().Sub(time.Now().Add(-1 * time.Hour * 24)),
			expected: "1 day ago",
		},
		{
			desc:     "hours ago",
			dur:      time.Now().Sub(time.Now().Add(-2 * time.Hour)),
			expected: "2 hours ago",
		},
		{
			desc:     "hour ago",
			dur:      time.Now().Sub(time.Now().Add(-1 * time.Hour)),
			expected: "1 hour ago",
		},
		{
			desc:     "minutes ago",
			dur:      time.Now().Sub(time.Now().Add(-2 * time.Minute)),
			expected: "2 minutes ago",
		},
		{
			desc:     "minute ago",
			dur:      time.Now().Sub(time.Now().Add(-1 * time.Minute)),
			expected: "1 minute ago",
		},
		{
			desc:     "seconds ago",
			dur:      time.Now().Sub(time.Now().Add(-2 * time.Second)),
			expected: "2 seconds ago",
		},
		{
			desc:     "second ago",
			dur:      time.Now().Sub(time.Now().Add(-1 * time.Second)),
			expected: "1 second ago",
		},
		{
			desc:     "nanosecond ago",
			dur:      time.Now().Sub(time.Now().Add(-1 * time.Nanosecond)),
			expected: "0 seconds ago",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.expected, ago(test.dur))
		})
	}
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
			expectedErr: "remote phoney could not be found",
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
