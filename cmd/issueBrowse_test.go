package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_issueBrowse(t *testing.T) {
	oldBrowse := browse
	defer func() { browse = oldBrowse }()

	browse = func(url string) error {
		require.Equal(t, "https://gitlab.com/zaquestion/test/issues/1", url)
		return nil
	}

	issueBrowseCmd.Run(nil, []string{"1"})
}
