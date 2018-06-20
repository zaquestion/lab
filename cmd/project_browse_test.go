package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_projectBrowse(t *testing.T) {
	oldBrowse := browse
	defer func() { browse = oldBrowse }()

	browse = func(url string) error {
		require.Equal(t, "https://gitlab.com/zaquestion/test", url)
		return nil
	}

	projectBrowseCmd.Run(nil, []string{""})
}
