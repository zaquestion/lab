package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_mrCmd(t *testing.T) {
	repo := copyTestRepo(t)
	var mrID string
	t.Run("prepare", func(t *testing.T) {
		cmd := exec.Command("sh", "-c", labBinaryPath+` mr list lab-testing | grep -m1 'mr title' | cut -c2- | awk '{print $1}' | xargs `+labBinaryPath+` mr lab-testing -d`)
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			//t.Fatal(err)
		}
	})
	t.Run("create", func(t *testing.T) {
		git := exec.Command("git", "checkout", "mrtest")
		git.Dir = repo
		b, err := git.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}

		cmd := exec.Command(labBinaryPath, "mr", "create", "lab-testing", "master",
			"-m", "mr title",
			"-m", "mr description",
			"-a", "lab-testing",
		)
		cmd.Dir = repo

		b, _ = cmd.CombinedOutput()
		out := string(b)
		t.Log(out)
		require.Contains(t, out, "https://gitlab.com/lab-testing/test/-/merge_requests")

		i := strings.Index(out, "/diffs\n")
		mrID = strings.TrimPrefix(out[:i], "https://gitlab.com/lab-testing/test/-/merge_requests/")
		t.Log(mrID)
	})
	t.Run("show", func(t *testing.T) {
		if mrID == "" {
			t.Skip("mrID is empty, create likely failed")
		}
		cmd := exec.Command(labBinaryPath, "mr", "show", "lab-testing", mrID)
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}

		out := string(b)
		outStripped := stripansi.Strip(out) // This is required because glamour adds a lot of ansi chars
		require.Contains(t, out, "Project: lab-testing/test\n")
		require.Contains(t, out, "Branches: mrtest->master\n")
		require.Contains(t, out, "Status: Open\n")
		require.Contains(t, out, "Assignee: lab-testing\n")
		require.Contains(t, out, fmt.Sprintf("#%s mr title", mrID))
		require.Contains(t, out, "===================================")
		require.Contains(t, outStripped, "mr description")
		require.Contains(t, out, fmt.Sprintf("WebURL: https://gitlab.com/lab-testing/test/-/merge_requests/%s", mrID))
	})
	t.Run("delete", func(t *testing.T) {
		if mrID == "" {
			t.Skip("mrID is empty, create likely failed")
		}
		cmd := exec.Command(labBinaryPath, "mr", "lab-testing", "-d", mrID)
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}
		require.Contains(t, string(b), fmt.Sprintf("Merge Request #%s closed", mrID))
	})
}

func Test_mrCmd_MR_description_and_options(t *testing.T) {
	repo := copyTestRepo(t)
	var (
		mrID      string
		commentID string
	)
	t.Run("prepare", func(t *testing.T) {
		cmd := exec.Command("sh", "-c", labBinaryPath+` mr list lab-testing | grep -m1 'Fancy Description' | cut -c2- | awk '{print $1}' | xargs `+labBinaryPath+` mr lab-testing -d`)
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			//t.Fatal(err)
		}
	})
	t.Run("create MR from file", func(t *testing.T) {
		git := exec.Command("git", "checkout", "mrtest")
		git.Dir = repo
		b, err := git.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}

		err = ioutil.WriteFile(filepath.Join(repo, "hellolab.txt"), []byte("Fancy Description\n\nFancy body of text describing this merge request.\n"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		cmd := exec.Command(labBinaryPath, "mr", "create", "lab-testing", "master",
			"-F", "hellolab.txt",
			"-a", "lab-testing",
		)
		cmd.Dir = repo

		b, _ = cmd.CombinedOutput()
		out := string(b)
		t.Log(out)
		require.Contains(t, out, "https://gitlab.com/lab-testing/test/-/merge_requests")

		i := strings.Index(out, "/diffs\n")
		mrID = strings.TrimPrefix(out[:i], "https://gitlab.com/lab-testing/test/-/merge_requests/")
		t.Log(mrID)

	})
	t.Run("update MR description", func(t *testing.T) {
		update := exec.Command(labBinaryPath, "mr", "edit", "lab-testing", mrID, "-m", "Updated Description", "-m", "Updated body of text describing this merge request.")
		update.Dir = repo
		b, err := update.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}
		cmd := exec.Command(labBinaryPath, "mr", "show", "lab-testing", mrID)
		cmd.Dir = repo
		b, err = cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}
		out := string(b)
		out = stripansi.Strip(out)

		require.Contains(t, out, "Updated Description")
		require.Contains(t, out, "Updated body of text describing this merge request.")
		require.NotContains(t, out, "Fancy")
	})
	t.Run("add MR comment", func(t *testing.T) {
		addComment := exec.Command(labBinaryPath, "mr", "note", "lab-testing", mrID, "-m", "Fancy comment on this merge request.")
		addComment.Dir = repo
		b, err := addComment.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}
		out := string(b)
		s := strings.Split(out, "_")
		commentID = s[2]
		s = strings.Split(commentID, "\n")
		commentID = s[0]

		t.Log("commentID =", commentID)

		url := "https://gitlab.com/lab-testing/test/merge_requests/" + mrID + "#note_" + commentID
		require.Contains(t, out, url)
	})
	t.Run("show MR with comment", func(t *testing.T) {
		showComment := exec.Command(labBinaryPath, "mr", "show", "lab-testing", mrID, "--comments")
		showComment.Dir = repo
		b, err := showComment.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}
		out := string(b)
		t.Log("commentID =", commentID)
		_commentID := "#" + commentID + ": lab-testing"
		require.Contains(t, out, _commentID)
	})
	t.Run("delete", func(t *testing.T) {
		if mrID == "" {
			t.Skip("mrID is empty, create -F likely failed")
		}
		cmd := exec.Command(labBinaryPath, "mr", "lab-testing", "-d", mrID)
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}
		require.Contains(t, string(b), fmt.Sprintf("Merge Request #%s closed", mrID))
	})
}

func Test_mrCmd_noArgs(t *testing.T) {
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "mr")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	require.Contains(t, string(b), `Usage:
  lab mr [flags]
  lab mr [command]`)
}

func Test_determineSourceRemote(t *testing.T) {
	tests := []struct {
		desc     string
		branch   string
		expected string
	}{
		{
			desc:     "branch.<name>.remote",
			branch:   "mrtest",
			expected: "lab-testing",
		},
		{
			desc:     "branch.<name>.pushRemote",
			branch:   "mrtest-pushRemote",
			expected: "lab-testing",
		},
		{
			desc:     "pushDefault without pushRemote set",
			branch:   "mrtest",
			expected: "garbageurl",
		},
		{
			desc:     "pushDefault with pushRemote set",
			branch:   "mrtest-pushRemote",
			expected: "lab-testing",
		},
	}

	// The function being tested here depends on being in the test
	// directory, where 'git config --local' can retrieve the correct
	// info from
	repo := copyTestRepo(t)
	oldWd, err := os.Getwd()
	if err != nil {
		t.Log(err)
	}
	os.Chdir(repo)

	var remoteModified bool
	for _, test := range tests {
		test := test
		if strings.Contains(test.desc, "pushDefault") && !remoteModified {
			git := exec.Command("git", "config", "--local", "remote.pushDefault", "garbageurl")
			git.Dir = repo
			b, err := git.CombinedOutput()
			if err != nil {
				t.Log(string(b))
				t.Fatal(err)
			}
			remoteModified = true
		}

		t.Run(test.desc, func(t *testing.T) {
			sourceRemote := determineSourceRemote(test.branch)
			assert.Equal(t, test.expected, sourceRemote)
		})
	}
	// Remove the added option to avoid messing with other tests
	git := exec.Command("git", "config", "--local", "--unset", "remote.pushDefault")
	git.Dir = repo
	b, err := git.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	// And move back to the workdir we were before the test
	os.Chdir(oldWd)
}
