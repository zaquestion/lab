package gitlab

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	err := os.Chdir(os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata"))
	if err != nil {
		log.Fatal(err)
	}
	os.Rename("test.git", ".git")
	code := m.Run()
	os.Rename(".git", "test.git")
	os.Exit(code)
}

func TestLoadGitLabTmplMR(t *testing.T) {
	mrTmpl := LoadGitLabTmpl(TmplMR)
	require.Equal(t, mrTmpl, "I am the mr tmpl")
}

func TestLoadGitLabTmpl(t *testing.T) {
	issueTmpl := LoadGitLabTmpl(TmplIssue)
	require.Equal(t, issueTmpl, "I am the issue tmpl")
}
