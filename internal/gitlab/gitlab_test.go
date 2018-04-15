package gitlab

import (
	"log"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	err := os.Chdir(os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata"))
	if err != nil {
		log.Fatal(err)
	}

	viper.SetConfigName("lab")
	viper.SetConfigType("hcl")
	viper.AddConfigPath(".")
	err = viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
	c := viper.AllSettings()["core"]
	config := c.([]map[string]interface{})[0]

	Init(
		config["host"].(string),
		config["user"].(string),
		config["token"].(string))
	os.Exit(m.Run())
}

func TestLoadGitLabTmplMR(t *testing.T) {
	mrTmpl := LoadGitLabTmpl(TmplMR)
	require.Equal(t, mrTmpl, "I am the default merge request template for lab")
}

func TestLoadGitLabTmplIssue(t *testing.T) {
	issueTmpl := LoadGitLabTmpl(TmplIssue)
	require.Equal(t, issueTmpl, "This is the default issue template for lab")
}

func TestLint(t *testing.T) {
	tests := []struct {
		desc     string
		content  string
		expected bool
	}{
		{
			"Valid",
			`build1:
  stage: build
  script:
    - echo "Do your build here"`,
			true,
		},
		{
			"Invalid",
			`build1:
    - echo "Do your build here"`,
			false,
		},
		{
			"Empty",
			``,
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			test := test
			t.Parallel()
			ok, _ := Lint(test.content)
			require.Equal(t, test.expected, ok)
		})
	}
}

func TestBranchPushed(t *testing.T) {
	tests := []struct {
		desc     string
		branch   string
		expected bool
	}{
		{
			"alpha is pushed",
			"mrtest",
			true,
		},
		{
			"needs encoding is pushed",
			"needs/encode",
			true,
		},
		{
			"alpha not pushed",
			"not_a_branch",
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			test := test
			t.Parallel()
			ok := BranchPushed(4181224, test.branch)
			require.Equal(t, test.expected, ok)
		})
	}
}
