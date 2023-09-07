package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_milestoneBrowse(t *testing.T) {
	oldBrowse := browse
	defer func() { browse = oldBrowse }()

	browse = func(url string) error {
		require.Equal(t, "https://gitlab.com/zaquestion/test/-/milestones/1", url)
		return nil
	}

	// milestone "1.0" has id 1
	milestoneBrowseCmd.Run(nil, []string{"1.0"})
}
