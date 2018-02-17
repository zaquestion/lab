package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_snippetCmd_personal(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	var snipID string
	t.Run("create_personal", func(t *testing.T) {
		cmd := exec.Command("../lab_bin", "snippet", "-g",
			"-m", "personal snippet title",
			"-m", "personal snippet description")
		cmd.Dir = repo

		rc, err := cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}

		_, err = rc.Write([]byte("personal snippet contents"))
		if err != nil {
			t.Fatal(err)
		}
		err = rc.Close()
		if err != nil {
			t.Fatal(err)
		}

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}

		out := string(b)
		require.Contains(t, out, "https://gitlab.com/snippets/")

		i := strings.Index(out, "\n")
		snipID = strings.TrimPrefix(out[:i], "https://gitlab.com/snippets/")
		t.Log(snipID)
	})
	t.Run("list_personal", func(t *testing.T) {
		// Issue: https://gitlab.com/gitlab-org/gitlab-ce/issues/43361
		t.Skip("borked")
		if snipID == "" {
			t.Skip("snipID is empty, create likely failed")
		}
		cmd := exec.Command("../lab_bin", "snippet", "-l", "-g")
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}

		snips := strings.Split(string(b), "\n")
		t.Log(snips)
		require.Contains(t, snips, fmt.Sprintf("#%s personal snippet title", snipID))
	})
	t.Run("delete_personal", func(t *testing.T) {
		if snipID == "" {
			t.Skip("snipID is empty, create likely failed")
		}
		cmd := exec.Command("../lab_bin", "snippet", "-g", "-d", snipID)
		cmd.Dir = repo

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}
		require.Contains(t, string(b), fmt.Sprintf("Snippet #%s deleted", snipID))
	})
}

func Test_snippetCmd_noArgs(t *testing.T) {
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "snippet")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	require.Contains(t, string(b), `Usage:
  lab snippet [flags]
  lab snippet [command]`)
}
