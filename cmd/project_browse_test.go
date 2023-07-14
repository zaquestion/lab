package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_projectBrowse(t *testing.T) {
	oldBrowse := browse
	defer func() { browse = oldBrowse }()

	browse = func(url string) error {
		require.Equal(t, "https://gitlab.com/zaquestion/test/-/blob/master", url)
		return nil
	}

	projectBrowseCmd.Run(nil, []string{""})
}

func Test_projectGetPath(t *testing.T) {
	defaultPath := projectBrowseGetPath("https://gitlab.com/zaquestion/test", "master", "", "")
	require.Equal(t, defaultPath, "https://gitlab.com/zaquestion/test/-/blob/master")
}

func Test_projectGetPathAndFile(t *testing.T) {
	pathAndFile := projectBrowseGetPath("https://gitlab.com/zaquestion/test", "master", "", "README.md")
	require.Equal(t, pathAndFile, "https://gitlab.com/zaquestion/test/-/blob/master/README.md")
}

func Test_projectGetPathAndFileAndBranch(t *testing.T) {
	pathAndFileAndBranch := projectBrowseGetPath("https://gitlab.com/zaquestion/test", "master", "new/branch", "README.md")
	require.Equal(t, pathAndFileAndBranch, "https://gitlab.com/zaquestion/test/-/blob/new/branch/README.md")
}

func Test_projectGetPathAndRef(t *testing.T) {
	pathRef := projectBrowseGetPath("https://gitlab.com/zaquestion/test", "", "12345abcdef", "")
	require.Equal(t, pathRef, "https://gitlab.com/zaquestion/test/-/blob/12345abcdef")
}
