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

	// This code is currently just helping me thinking about how implement a test
	// for this PR behavior
	cmd := git.New("branch")
	cmd.Stdout = nil
	gBranches, err := cmd.Output()
	if err != nil {
		return nil
	}
	branches := strings.Split(string(gBranches), "\n")
	for _, b := range branches {
		fmt.Println(b)
	}
	return nil
}
