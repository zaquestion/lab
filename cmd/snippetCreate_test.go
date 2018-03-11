package cmd

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_snippetCreate(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "snippet", "create", "lab-testing",
		"-m", "snippet title",
		"-m", "snippet description")
	cmd.Dir = repo

	rc, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}

	_, err = rc.Write([]byte("snippet contents"))
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

	require.Contains(t, string(b), "https://gitlab.com/lab-testing/test/snippets/")
}

func Test_snippetCreate_file(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	t.Log(filepath.Join(repo, "testfile"))
	err := ioutil.WriteFile(filepath.Join(repo, "testfile"), []byte("test file contents"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("../lab_bin", "snippet", "lab-testing", "testfile",
		"-m", "snippet title",
		"-m", "snippet description")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	require.Contains(t, string(b), "https://gitlab.com/lab-testing/test/snippets/")
}

func Test_snippetCreate_Global(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)

	cmd := exec.Command("../lab_bin", "snippet", "create", "-g",
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

	require.Contains(t, string(b), "https://gitlab.com/snippets/")
}

// This test is a little ridiculus, if we find it doesn't work well on other
// envionments, we can just remove it. Its sole purpose is to test that a
// personal snippet can be created (with the users git editor) outside of a git
// repo. issue #98
func Test_snippetCreate_Global_Editor(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)

	err := exec.Command("cp", "-rf", repo, "/tmp/testdata-6810350901254661225").Run()
	if err != nil {
		t.Fatal(err)
	}
	repo = "/tmp/testdata-6810350901254661225"
	defer func() {
		coveragePath, _ := filepath.Glob("/tmp/coverage-*")
		exec.Command("cp", coveragePath[0], "../coverage-6810350901254661225.out").Run()
		os.Remove(coveragePath[0])
		os.RemoveAll(repo)
	}()

	// Write the editor file here, since its tricky to get a file with
	// contents in it otherwise. We need a file with contents to
	// successfully create the snippet
	err = ioutil.WriteFile("/tmp/SNIPCODE_EDITMSG", []byte("personal snippet title outside repo"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Remove .git dir forcing the cmd to exec outside of a git repo
	cmd := exec.Command("rm", "-rf", ".git")
	cmd.Dir = repo
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	cmd = exec.Command(os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/lab_bin"), "snippet", "create", "-g")
	cmd.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "EDITOR=test -f"}
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	require.Contains(t, string(b), "https://gitlab.com/snippets/")
}

func Test_snipMsg(t *testing.T) {
	msgs, err := snippetCreateCmd.Flags().GetStringSlice("message")
	if err != nil {
		t.Fatal(err)
	}
	title, desc := snipMsg(msgs)
	assert.Equal(t, "-", title)
	assert.Equal(t, "", desc)
}

func Test_snipCode(t *testing.T) {
	err := ioutil.WriteFile("./testfile", []byte("test file contents"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		Name         string
		Path         string
		ExpectedCode string
	}{
		{
			Name:         "From File",
			Path:         "./testfile",
			ExpectedCode: "test file contents",
		},
		{
			Name:         "From Editor",
			Path:         "",
			ExpectedCode: "\n\n",
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			code, err := snipCode(test.Path)
			if err != nil {
				t.Fatal(err)
			}
			require.Equal(t, test.ExpectedCode, code)
		})
	}
}

func Test_snipText(t *testing.T) {
	var tmpl = "foo" + `
{{.CommentChar}} In this mode you are writing a snippet from scratch
{{.CommentChar}} The first block is the title and the rest is the contents.`
	text, err := snipText(tmpl)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, `foo
# In this mode you are writing a snippet from scratch
# The first block is the title and the rest is the contents.`, text)

}
