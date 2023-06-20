package cmd

import (
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_token(t *testing.T) {
	repo := copyTestRepo(t)

	now := time.Now()
	expiryString := fmt.Sprintf("%d-%d-%d", now.Year(), now.Month(), now.Day() + 1)

	t.Run("create", func(t *testing.T) {
		cmd := exec.Command(labBinaryPath, "token", "create", "--name", now.String(), "--expiresat", expiryString, "--scopes", "read_api")
		cmd.Dir = repo

		cmdOut, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(cmdOut))
			t.Fatal(err)
		}
		require.Contains(t, string(cmdOut), now.String())
	})

	t.Run("list", func(t *testing.T) {
		cmd2 := exec.Command(labBinaryPath, "token", "list")
		cmd2.Dir = repo

		cmd2Out, err := cmd2.CombinedOutput()
		if err != nil {
			t.Log(string(cmd2Out))
			t.Fatal(err)
		}
		require.Contains(t, string(cmd2Out), now.String())
	})

	t.Run("revoke", func(t *testing.T) {
		cmd3 := exec.Command(labBinaryPath, "token", "revoke", now.String())
		cmd3.Dir = repo

		cmd3Out, err := cmd3.CombinedOutput()
		if err != nil {
			t.Log(string(cmd3Out))
			t.Fatal(err)
		}
		require.Contains(t, string(cmd3Out), now.String())
	})

	t.Run("list after revoke", func(t *testing.T) {
		cmd4 := exec.Command(labBinaryPath, "token", "list")
		cmd4.Dir = repo

		cmd4Out, err := cmd4.CombinedOutput()
		if err != nil {
			t.Log(string(cmd4Out))
			t.Fatal(err)
		}
		require.NotContains(t, string(cmd4Out), now.String())
	})
}
