package config

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	testconf := "/tmp/testconf-" + strconv.Itoa(int(rand.Uint64()))
	os.Mkdir(testconf, os.FileMode(0700))

	t.Run("create config", func(t *testing.T) {
		old := os.Stdout // keep backup of the real stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		var buf bytes.Buffer
		fmt.Fprintln(&buf, "https://gitlab.zaquestion.io")
		fmt.Fprintln(&buf, "zaq")

		oldreadPassword := readPassword
		readPassword = func() (string, error) {
			return "abcde12345", nil
		}
		defer func() {
			readPassword = oldreadPassword
		}()

		err := New(path.Join(testconf, "lab.hcl"), &buf)
		if err != nil {
			t.Fatal(err)
		}

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

		assert.Contains(t, out, "Enter default GitLab host (default: https://gitlab.com): ")
		assert.Contains(t, out, "Enter default GitLab user:")
		assert.Contains(t, out, "Create a token here: https://gitlab.zaquestion.io/profile/personal_access_tokens\nEnter default GitLab token (scope: api):")

		cfg, err := os.Open(path.Join(testconf, "lab.hcl"))
		if err != nil {
			t.Fatal(err)
		}

		cfgData, err := ioutil.ReadAll(cfg)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, string(cfgData), `"core" = {
  "host" = "https://gitlab.zaquestion.io"

  "token" = "abcde12345"

  "user" = "zaq"
}`)
	})
	os.RemoveAll(testconf)
}
