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
	os.Exit(m.Run())
}

func TestLoadGitLabTmplMR(t *testing.T) {
	mrTmpl := LoadGitLabTmpl(TmplMR)
	require.Equal(t, mrTmpl, "I am the mr tmpl")
}

func TestLoadGitLabTmplIssue(t *testing.T) {
	issueTmpl := LoadGitLabTmpl(TmplIssue)
	require.Equal(t, issueTmpl, "I am the issue tmpl")
}
