package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_token(t *testing.T) {
	repo := copyTestRepo(t)

	// As of 16.1, token creation is limited to administrators.  Without
	// creating a token it is difficult to test the creation and revoking
	// of a token.  If GitLab changes the permissions on creating tokens
	// then the commit that introduced this message can be reverted.
	t.Run("list", func(t *testing.T) {
		cmd2 := exec.Command(labBinaryPath, "token", "list")
		cmd2.Dir = repo

		cmd2Out, err := cmd2.CombinedOutput()
		if err != nil {
			t.Log(string(cmd2Out))
			t.Fatal(err)
		}
		require.NotEmpty(t, string(cmd2Out))
	})
}
