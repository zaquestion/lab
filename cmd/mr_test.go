package cmd

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/require"
)

func closeMR(t *testing.T, targetRepo string, cmdDir string, mrID string) {
	if mrID == "" {
		t.Skip("mrID is empty, create likely failed")
	}
	cmd := exec.Command(labBinaryPath, "mr", "close", targetRepo, mrID)
	cmd.Dir = cmdDir

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	require.Contains(t, string(b), fmt.Sprintf("Merge Request !%s closed", mrID))
}

func cleanupMR(t *testing.T, targetRepo string, cmdDir string, MRtitle string) {
	var openMRcmd *exec.Cmd

	if MRtitle == "" {
		openMRcmd = exec.Command(labBinaryPath, "mr", "list", targetRepo)
	} else {
		openMRcmd = exec.Command(labBinaryPath, "mr", "list", targetRepo, MRtitle)
	}
	openMRcmd.Dir = cmdDir
	openMRout, err := openMRcmd.CombinedOutput()
	if err != nil {
		t.Log(string(openMRout))
	}

	// find MR number
	s := strings.Split(string(openMRout), " ")
	openMRstr := s[0]
	// strip off "!"
	openMRstr = openMRstr[1:]

	openMR, err := strconv.Atoi(openMRstr)
	if err != nil {
		t.Log(string(openMRstr))
		return
	}

	if openMR <= 0 {
		// no open MRs
		return
	}

	// close the existing MR
	closeMR(t, targetRepo, cmdDir, string(openMRstr))
}

func Test_mrCmd(t *testing.T) {
	repo := copyTestRepo(t)
	var mrID string
	t.Run("prepare", func(t *testing.T) {
		cleanupMR(t, "lab-testing", repo, "")
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
		require.Contains(t, out, fmt.Sprintf("!%s mr title", mrID))
		require.Contains(t, out, "===================================")
		require.Contains(t, outStripped, "mr description")
		require.Contains(t, out, fmt.Sprintf("WebURL: https://gitlab.com/lab-testing/test/-/merge_requests/%s", mrID))
	})
	t.Run("close", func(t *testing.T) {
		closeMR(t, "lab-testing", repo, mrID)
	})
}

func Test_mrCmd_MR_description_and_options(t *testing.T) {
	repo := copyTestRepo(t)
	var (
		mrID      string
		commentID string
	)
	t.Run("prepare", func(t *testing.T) {
		cleanupMR(t, "lab-testing", repo, "")
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
	t.Run("close", func(t *testing.T) {
		closeMR(t, "lab-testing", repo, mrID)
	})
}

func Test_mrCmd_DifferingUpstreamBranchName(t *testing.T) {
	repo := copyTestRepo(t)
	var mrID string
	t.Run("prepare", func(t *testing.T) {
		cleanupMR(t, "lab-testing", repo, "")
	})
	t.Run("create", func(t *testing.T) {
		git := exec.Command("git", "checkout", "-b", "local/mrtest", "origin/mrtest")
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
	t.Run("close", func(t *testing.T) {
		closeMR(t, "lab-testing", repo, mrID)
	})
}

func Test_mrCmd_Draft(t *testing.T) {
	repo := copyTestRepo(t)
	var mrID string
	t.Run("prepare", func(t *testing.T) {
		cleanupMR(t, "lab-testing", repo, "")
	})
	t.Run("create", func(t *testing.T) {
		git := exec.Command("git", "checkout", "mrtest")
		git.Dir = repo
		b, err := git.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}

		cmd := exec.Command(labBinaryPath, "mr", "create", "--draft", "lab-testing", "master",
			"-m", "Test draft merge request",
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
	t.Run("list", func(t *testing.T) {
		if mrID == "" {
			t.Skip("mrID is empty, create likely failed")
		}
		cmd := exec.Command(labBinaryPath, "mr", "list", "--draft", "lab-testing")
		cmd.Dir = repo

		b, _ := cmd.CombinedOutput()
		out := string(b)
		t.Log(out)
		require.Contains(t, out, "Test draft merge request")
	})
	t.Run("modify", func(t *testing.T) {
		if mrID == "" {
			t.Skip("mrID is empty, create likely failed")
		}
		cmd := exec.Command(labBinaryPath, "mr", "edit", "--ready", "lab-testing")
		cmd.Dir = repo

		b, _ := cmd.CombinedOutput()
		t.Log(string(b))

		cmd = exec.Command(labBinaryPath, "mr", "list", "--draft", "lab-testing")
		cmd.Dir = repo

		b, _ = cmd.CombinedOutput()
		out := string(b)
		t.Log(out)
		require.NotContains(t, out, "Test draft merge request")
	})
	t.Run("close", func(t *testing.T) {
		closeMR(t, "lab-testing", repo, mrID)
	})
}

func Test_mrCmd_Milestone(t *testing.T) {
	repo := copyTestRepo(t)
	var mrID string
	t.Run("prepare", func(t *testing.T) {
		cleanupMR(t, "origin", repo, "Test draft")
	})
	t.Run("create", func(t *testing.T) {
		git := exec.Command("git", "checkout", "mrtest")
		git.Dir = repo
		b, err := git.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}

		cmd := exec.Command(labBinaryPath, "mr", "create", "--milestone", "1.0", "origin", "master",
			"-m", "MR for 1.0",
		)
		cmd.Dir = repo

		b, _ = cmd.CombinedOutput()
		out := string(b)
		t.Log(out)
		require.Contains(t, out, "https://gitlab.com/zaquestion/test/-/merge_requests")

		i := strings.Index(out, "/diffs\n")
		mrID = strings.TrimPrefix(out[:i], "https://gitlab.com/zaquestion/test/-/merge_requests/")
		t.Log(mrID)
	})
	t.Run("list", func(t *testing.T) {
		if mrID == "" {
			t.Skip("mrID is empty, create likely failed")
		}
		cmd := exec.Command(labBinaryPath, "mr", "list", "--milestone", "1.0", "origin")
		cmd.Dir = repo

		b, _ := cmd.CombinedOutput()
		out := string(b)
		t.Log(out)
		require.Contains(t, out, "MR for 1.0")
	})
	t.Run("modify", func(t *testing.T) {
		if mrID == "" {
			t.Skip("mrID is empty, create likely failed")
		}
		cmd := exec.Command(labBinaryPath, "mr", "edit", "--milestone", "", "origin")
		cmd.Dir = repo

		b, _ := cmd.CombinedOutput()
		t.Log(string(b))

		cmd = exec.Command(labBinaryPath, "mr", "list", "--milestone", "1.0", "origin")
		cmd.Dir = repo

		b, _ = cmd.CombinedOutput()
		out := string(b)
		t.Log(out)
		require.NotContains(t, out, "MR for 1.0")
	})
	t.Run("close", func(t *testing.T) {
		closeMR(t, "origin", repo, mrID)
	})
}

func Test_mrCmd_ByBranch(t *testing.T) {
	repo := copyTestRepo(t)
	var mrID string
	t.Run("prepare", func(t *testing.T) {
		cleanupMR(t, "lab-testing", repo, "")
	})
	t.Run("create", func(t *testing.T) {
		git := exec.Command("git", "checkout", "mrtest")
		git.Dir = repo
		b, err := git.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}

		cmd := exec.Command(labBinaryPath, "mr", "create", "--draft", "lab-testing", "master",
			"-m", "mr by branch",
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
		cmd := exec.Command(labBinaryPath, "mr", "show", "lab-testing", "mrtest")
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}

		out := string(b)
		require.Contains(t, out, fmt.Sprintf("WebURL: https://gitlab.com/lab-testing/test/-/merge_requests/%s", mrID))
	})
	t.Run("close", func(t *testing.T) {
		closeMR(t, "lab-testing", repo, mrID)
	})
}

func Test_mrCmd_source(t *testing.T) {
	repo := copyTestRepo(t)
	var mrID string
	t.Run("prepare", func(t *testing.T) {
		cleanupMR(t, "lab-testing", repo, "")
	})
	t.Run("create_invalid", func(t *testing.T) {
		git := exec.Command("git", "checkout", "mrtest")
		git.Dir = repo
		b, err := git.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}

		cmd := exec.Command(labBinaryPath, "mr", "create", "lab-testing", "master",
			"--source", "origin:mrtestDoesNotExist",
			"-m", "mr title",
			"-m", "mr description",
			"-a", "lab-testing",
		)
		cmd.Dir = repo

		b, _ = cmd.CombinedOutput()
		out := string(b)
		t.Log(out)
		require.Contains(t, out, "Aborting MR create, origin:mrtestDoesNotExist is not a valid reference")
	})
	t.Run("create_valid", func(t *testing.T) {
		git := exec.Command("git", "checkout", "mrtest")
		git.Dir = repo
		b, err := git.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}

		cmd := exec.Command(labBinaryPath, "mr", "create", "lab-testing", "master",
			"--source", "origin:mrtest",
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
	t.Run("close", func(t *testing.T) {
		closeMR(t, "lab-testing", repo, mrID)
	})
}

func Test_mrCmd_assign_and_review(t *testing.T) {
	var mrID string
	mrIDString := "MR for assign and review commands"

	repo := copyTestRepo(t)

	// find the test MR
	mrIDlist := exec.Command(labBinaryPath, "mr", "list", mrIDString)
	mrIDlist.Dir = repo
	mrIDOut, err := mrIDlist.CombinedOutput()
	if err != nil {
		t.Log(string(mrIDOut))
		t.Fatal(err)
	}

	// find MR number
	s := strings.Split(string(mrIDOut), " ")
	mrID = s[0]
	// strip off "!"
	mrID = mrID[1:]

	mrURL := "https://gitlab.com/zaquestion/test/-/merge_requests/" + mrID

	t.Run("assign_and_unassign", func(t *testing.T) {
		mrEdit := exec.Command(labBinaryPath, "mr", "edit", mrID, "--assign", "lab-testing")
		mrEdit.Dir = repo
		mrEditOut, err := mrEdit.CombinedOutput()
		if err != nil {
			t.Log(string(mrEditOut))
			t.Fatal(err)
		}

		mrList := exec.Command(labBinaryPath, "mr", "list", "--assignee", "lab-testing")
		mrList.Dir = repo
		mrListOut, err := mrList.CombinedOutput()
		if err != nil {
			t.Log(string(mrListOut))
			t.Fatal(err)
		}

		mrUnEdit := exec.Command(labBinaryPath, "mr", "edit", mrID, "--unassign", "lab-testing")
		mrUnEdit.Dir = repo
		mrUnEditOut, err := mrUnEdit.CombinedOutput()
		if err != nil {
			t.Log(string(mrUnEditOut))
			t.Fatal(err)
		}

		require.Contains(t, string(mrEditOut), mrURL)
		require.Contains(t, string(mrListOut), mrIDString)
		require.Contains(t, string(mrUnEditOut), mrURL)
	})

	// This tests 'lab mr edit --review' and 'lab mr list --unreview'
	t.Run("review_and_unreview", func(t *testing.T) {
		mrEdit := exec.Command(labBinaryPath, "mr", "edit", mrID, "--review", "lab-testing")
		mrEdit.Dir = repo
		mrEditOut, err := mrEdit.CombinedOutput()
		if err != nil {
			t.Log(string(mrEditOut))
			t.Fatal(err)
		}

		mrList := exec.Command(labBinaryPath, "mr", "list", "--reviewer", "lab-testing")
		mrList.Dir = repo
		mrListOut, err := mrList.CombinedOutput()
		if err != nil {
			t.Log(string(mrListOut))
			t.Fatal(err)
		}

		mrUnEdit := exec.Command(labBinaryPath, "mr", "edit", mrID, "--unreview", "lab-testing")
		mrUnEdit.Dir = repo
		mrUnEditOut, err := mrUnEdit.CombinedOutput()
		if err != nil {
			t.Log(string(mrUnEditOut))
			t.Fatal(err)
		}

		require.Contains(t, string(mrEditOut), mrURL)
		require.Contains(t, string(mrListOut), mrIDString)
		require.Contains(t, string(mrUnEditOut), mrURL)

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
