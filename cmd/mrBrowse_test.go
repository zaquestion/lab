package cmd

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func Test_mrBrowse(t *testing.T) {
	viper.SetConfigName("lab")
	viper.SetConfigType("hcl")
	viper.AddConfigPath("../testdata")
	err := viper.ReadInConfig()
	if err != nil {
		t.Error(err)
	}

	oldBrowse := browse
	defer func() { browse = oldBrowse }()

	browse = func(url string) error {
		require.Equal(t, "https://gitlab.com/zaquestion/test/merge_requests/1", url)
		return nil
	}

	mrBrowseCmd.Run(nil, []string{"1"})
}
