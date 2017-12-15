package cmd

import (
	"io/ioutil"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_snippetCreate(t *testing.T) {
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

func Test_snippetCreate_Global(t *testing.T) {
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

func Test_snipMsg(t *testing.T) {
	title, desc, err := snipMsg(nil, "snip title\nthis should be dropped")
	if err != nil {
		t.Fatal(err)
	}
	// This title was defaulted from the snippet contents/code because no
	// msgs -m title was provided
	assert.Equal(t, "snip title", title)
	// This is the body created in during editing or with provided msgs -m
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
