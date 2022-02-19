package cmd

import (
	"bufio"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_mrCreateDiscussion(t *testing.T) {
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr", "discussion", "lab-testing", mrCommentSlashDiscussionDumpsterID,
		"-m", "discussion text")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	require.Contains(t, string(b), "https://gitlab.com/lab-testing/test/merge_requests/"+mrCommentSlashDiscussionDumpsterID+"#note_")
}

func Test_mrCreateDiscussion_file(t *testing.T) {
	repo := copyTestRepo(t)

	err := ioutil.WriteFile(filepath.Join(repo, "hellolab.txt"), []byte("hello\nlab\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(labBinaryPath, "mr", "discussion", "lab-testing", mrCommentSlashDiscussionDumpsterID,
		"-F", "hellolab.txt")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	require.Contains(t, string(b), "https://gitlab.com/lab-testing/test/merge_requests/"+mrCommentSlashDiscussionDumpsterID+"#note_")
}

func Test_mrDiscussionMsg(t *testing.T) {
	tests := []struct {
		Name         string
		Msgs         []string
		ExpectedBody string
	}{
		{
			Name:         "Using messages",
			Msgs:         []string{"discussion paragraph 1", "discussion paragraph 2"},
			ExpectedBody: "discussion paragraph 1\n\ndiscussion paragraph 2",
		},
		{
			Name:         "From Editor",
			Msgs:         nil,
			ExpectedBody: "", // this is not a great test
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			body, err := mrDiscussionMsg(1, "OPEN", "", test.Msgs, "")
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, test.ExpectedBody, body)
		})
	}
}

func Test_mrDiscussionText(t *testing.T) {
	t.Parallel()
	tmpl := mrDiscussionGetTemplate("")
	text, err := noteText(1701, "OPEN", "", "\n", tmpl)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, `

# This thread is being started on OPEN Merge Request 1701.
# Comment lines beginning with '#' are discarded.`, text)
}

// test !1159 is reserved for testing commit comments
var mrDiscussionCommitMRid = "1159"

func mrDiscussionShow(t *testing.T, repo string) string {
	cmd := exec.Command(labBinaryPath, "mr", "show", mrDiscussionCommitMRid, "--comments", "--no-pager")
	cmd.Dir = repo

	a, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(a))
		t.Fatal(err)
	}

	return string(a)
}

func mrDiscussionCommitCommentCleanup(t *testing.T, repo string) {

	showText := mrDiscussionShow(t, repo)

	scanner := bufio.NewScanner(strings.NewReader(showText))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#") {
			continue
		}
		// lab mr edit 1159:<id> --delete-note
		commentid := strings.Split(line, ": ")[0][1:]
		cmd := exec.Command(labBinaryPath, "mr", "edit", mrDiscussionCommitMRid+":"+commentid, "--delete-note")
		cmd.Dir = repo

		a, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(a))
			t.Fatal(err)
		}
	}
}

func Test_mrDiscussionCommitComments(t *testing.T) {
	var testCommitID = "538c74715661e04fd4ffffcb72a04b6da969f8dd"

	repo := copyTestRepo(t)

	mrDiscussionCommitCommentCleanup(t, repo)

	showText := mrDiscussionShow(t, repo)
	require.NotContains(t, showText, "old line 3")
	require.NotContains(t, showText, "context line 5")
	require.NotContains(t, showText, "new line 8")

	cmd := exec.Command(labBinaryPath, "mr", "checkout", mrDiscussionCommitMRid)
	cmd.Dir = repo

	a, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(a))
		t.Fatal(err)
	}

	//lab mr discussion 1159 --commit 538c74715661e04fd4ffffcb72a04b6da969f8dd --position=test:-3,3 -m "old line 3"
	cmd = exec.Command(labBinaryPath, "mr", "discussion", mrDiscussionCommitMRid, "--commit", testCommitID, "--position=test:-3,3", "-m", "old line 3")
	cmd.Dir = repo

	a, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(a))
		t.Fatal(err)
	}

	showText = mrDiscussionShow(t, repo)
	require.Contains(t, showText, "old line 3")
	require.Contains(t, showText, `
commit:538c74715661e04fd4ffffcb72a04b6da969f8dd
File:test
|        @@ -1,9 +1,11 @@
|  1   1  line 1
|  2   2  line 2
|  3     -line 3`)
	require.NotContains(t, showText, "context line 5")
	require.NotContains(t, showText, "new line 8")

	//lab mr discussion 1159 --commit 538c74715661e04fd4ffffcb72a04b6da969f8dd --position=test:\ 5,4 -m "context line 5"
	cmd = exec.Command(labBinaryPath, "mr", "discussion", mrDiscussionCommitMRid, "--commit", testCommitID, "--position=test: 5,4", "-m", "context line 5")
	cmd.Dir = repo

	a, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(a))
		t.Fatal(err)
	}

	showText = mrDiscussionShow(t, repo)
	require.Contains(t, showText, "context line 5")
	require.Contains(t, showText, `
commit:538c74715661e04fd4ffffcb72a04b6da969f8dd
File:test
|        @@ -1,9 +1,11 @@
|  1   1  line 1
|  2   2  line 2
|  3     -line 3
|  4   3  line 4
|  5   4  line 5`)
	require.NotContains(t, showText, "new line 8")

	//lab mr discussion 1159 --commit 538c74715661e04fd4ffffcb72a04b6da969f8dd --position=test:+8,8 -m "new line 8"
	cmd = exec.Command(labBinaryPath, "mr", "discussion", mrDiscussionCommitMRid, "--commit", testCommitID, "--position=test:+8,8", "-m", "new line 8")
	cmd.Dir = repo

	a, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(a))
		t.Fatal(err)
	}

	showText = mrDiscussionShow(t, repo)
	require.Contains(t, showText, "new line 8")
	require.Contains(t, showText, `
commit:538c74715661e04fd4ffffcb72a04b6da969f8dd
File:test
|  5   4  line 5
|  6   5  line 6
|  7   6  line 7
|  8     -line 8
|  9   7  line 9
|      8 +line 10`)
	mrDiscussionCommitCommentCleanup(t, repo)
}
