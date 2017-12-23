package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_mrBrowse(t *testing.T) {
	oldBrowse := browse
	defer func() { browse = oldBrowse }()

	browse = func(url string) error {
		require.Equal(t, "https://gitlab.com/zaquestion/test/merge_requests/1", url)
		return nil
	}

	mrBrowseCmd.Run(nil, []string{"1"})
}
