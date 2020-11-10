package cmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zaquestion/lab/internal/git"
)

// Make sure the version command don't break things in the future
func Test_versionCmd(t *testing.T) {
	git_cmd := git.New("version")
	git_cmd.Stdout = nil
	out, err := git_cmd.Output()
	if err != nil {
		t.Log(string(out))
		t.Fatal(err)
	}
	git_ver := strings.TrimSpace(string(out))

	t.Run("version", func(t *testing.T) {
		lab_cmd := exec.Command(labBinaryPath, "version")
		out, err := lab_cmd.CombinedOutput()
		if err != nil {
			t.Log(string(out))
			t.Fatal(err)
		}
		combined_ver := string(out)
		progs_ver := strings.Split(combined_ver, "\n")

		assert.Contains(t, progs_ver[0], git_ver)
		assert.Contains(t, progs_ver[1], "lab version "+Version)
	})
	t.Run("--version", func(t *testing.T) {
		old := os.Stdout // keep backup of the real stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		RootCmd.Flag("version").Value.Set("true")
		RootCmd.Run(RootCmd, nil)

		outC := make(chan string)
		// copy the output in a separate goroutine so printing can't block indefinitely
		go func() {
			var buf bytes.Buffer
			io.Copy(&buf, r)
			outC <- buf.String()
		}()

		// back to normal state
		w.Close()
		os.Stdout = old // restoring the real stdout
		out := <-outC

		assert.Contains(t, out, git_ver)
		assert.Contains(t, out, "lab version "+Version)
	})
}
