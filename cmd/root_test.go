package cmd

import (
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

func TestMain(m *testing.M) {
	wd, err := git.WorkingDir()
	if err != nil {
		log.Fatal(err)
	}
	os.Chdir(wd)
	err = exec.Command("go", "test", "-c", "-coverpkg", "./...", "-covermode", "count", "-o", "lab_bin").Run()
	if err != nil {
		log.Fatal(err)
	}
	rand.Seed(time.Now().UnixNano())

	// Load config for non-testbinary based tests
	os.Chdir(path.Join(wd, "testdata"))
	viper.SetConfigName("lab")
	viper.SetConfigType("hcl")
	viper.AddConfigPath(".")
	err = viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
	c := viper.AllSettings()["core"]
	config := c.([]map[string]interface{})[0]
	lab.Init(
		config["host"].(string),
		config["user"].(string),
		config["token"].(string))

	code := m.Run()
	os.Chdir(wd)
	os.Remove("lab_bin")
	testdirs, err := filepath.Glob("testdata-*")
	if err != nil {
		log.Fatal(err)
	}
	for _, dir := range testdirs {
		err := os.RemoveAll(dir)
		if err != nil {
			log.Fatal(err)
		}
	}

	os.Exit(code)
}

func copyTestRepo(t *testing.T) string {
	dir := "../testdata-" + strconv.Itoa(int(rand.Uint64()))
	t.Log(dir)
	err := exec.Command("cp", "-r", "../testdata", dir).Run()
	if err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir = path.Join(wd, dir)
	return dir
}

func TestRootCloneNoArg(t *testing.T) {
	cmd := exec.Command("../lab_bin", "clone")
	b, _ := cmd.CombinedOutput()
	require.Contains(t, string(b), "You must specify a repository to clone.")
}

func TestRootGitCmd(t *testing.T) {
	cmd := exec.Command("../lab_bin", "log", "-n", "1")
	b, _ := cmd.CombinedOutput()
	require.Contains(t, string(b), `commit 09b519cba018b707c98fc56e37df15806d89d866
Author: Zaq? Wiedmann <zaquestion@gmail.com>
Date:   Sun Apr 1 19:40:47 2018 -0700

    (ci) jobs with interleaved sleeps and prints`)
}

func TestRootNoArg(t *testing.T) {
	cmd := exec.Command("../lab_bin")
	b, _ := cmd.CombinedOutput()
	assert.Contains(t, string(b), "usage: git [--version] [--help] [-C <path>] [-c name=value]")
	assert.Contains(t, string(b), `These GitLab commands are provided by lab:

  fork          Fork a remote repository on GitLab and add as remote`)
}

func Test_parseArgsRemote(t *testing.T) {
	tests := []struct {
		Name           string
		Args           []string
		ExpectedString string
		ExpectedInt    int64
		ExpectedErr    string
	}{
		{
			Name:           "No Args",
			Args:           nil,
			ExpectedString: "",
			ExpectedInt:    0,
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg remote",
			Args:           []string{"origin"},
			ExpectedString: "origin",
			ExpectedInt:    0,
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg non remote",
			Args:           []string{"foo"},
			ExpectedString: "",
			ExpectedInt:    0,
			ExpectedErr:    "foo is not a valid remote or number",
		},
		{
			Name:           "1 arg page",
			Args:           []string{"100"},
			ExpectedString: "",
			ExpectedInt:    100,
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg invalid page",
			Args:           []string{"asdf100"},
			ExpectedString: "",
			ExpectedInt:    0,
			ExpectedErr:    "asdf100 is not a valid remote or number",
		},
		{
			Name:           "2 arg remote page",
			Args:           []string{"origin", "100"},
			ExpectedString: "origin",
			ExpectedInt:    100,
			ExpectedErr:    "",
		},
		{
			Name:           "2 arg invalid remote valid page",
			Args:           []string{"foo", "100"},
			ExpectedString: "",
			ExpectedInt:    0,
			ExpectedErr:    "foo is not a valid remote",
		},
		{
			Name:           "2 arg valid remote invalid page",
			Args:           []string{"foo", "asdf100"},
			ExpectedString: "",
			ExpectedInt:    0,
			ExpectedErr:    "strconv.ParseInt: parsing \"asdf100\": invalid syntax",
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			s, i, err := parseArgsRemote(test.Args)
			if err != nil {
				assert.EqualError(t, err, test.ExpectedErr)
			}
			assert.Equal(t, test.ExpectedString, s)
			assert.Equal(t, test.ExpectedInt, i)
		})
	}
}

func Test_parseArgs(t *testing.T) {
	tests := []struct {
		Name           string
		Args           []string
		ExpectedString string
		ExpectedInt    int64
		ExpectedErr    string
	}{
		{
			Name:           "No Args",
			Args:           nil,
			ExpectedString: "zaquestion/test",
			ExpectedInt:    0,
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg remote",
			Args:           []string{"lab-testing"},
			ExpectedString: "lab-testing/test",
			ExpectedInt:    0,
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg non remote",
			Args:           []string{"foo"},
			ExpectedString: "",
			ExpectedInt:    0,
			ExpectedErr:    "foo is not a valid remote or number",
		},
		{
			Name:           "1 arg page",
			Args:           []string{"100"},
			ExpectedString: "zaquestion/test",
			ExpectedInt:    100,
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg invalid page",
			Args:           []string{"asdf100"},
			ExpectedString: "",
			ExpectedInt:    0,
			ExpectedErr:    "asdf100 is not a valid remote or number",
		},
		{
			Name:           "2 arg remote page",
			Args:           []string{"origin", "100"},
			ExpectedString: "zaquestion/test",
			ExpectedInt:    100,
			ExpectedErr:    "",
		},
		{
			Name:           "2 arg invalid remote valid page",
			Args:           []string{"foo", "100"},
			ExpectedString: "",
			ExpectedInt:    0,
			ExpectedErr:    "foo is not a valid remote",
		},
		{
			Name:           "2 arg valid remote invalid page",
			Args:           []string{"foo", "asdf100"},
			ExpectedString: "",
			ExpectedInt:    0,
			ExpectedErr:    "strconv.ParseInt: parsing \"asdf100\": invalid syntax",
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			s, i, err := parseArgs(test.Args)
			if err != nil {
				assert.EqualError(t, err, test.ExpectedErr)
			}
			assert.Equal(t, test.ExpectedString, s)
			assert.Equal(t, test.ExpectedInt, i)
		})
	}
}
