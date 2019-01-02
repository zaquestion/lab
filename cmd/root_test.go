package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zaquestion/lab/internal/copy"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var labBinaryPath string

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	// Build a lab binary with test symbols. If the parent test binary was run
	// with coverage enabled, enable coverage on the child binary, too.
	var err error
	labBinaryPath, err = filepath.Abs(os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata/" + labBinary))
	if err != nil {
		log.Fatal(err)
	}
	testCmd := []string{"test", "-c", "-o", labBinaryPath, "github.com/zaquestion/lab"}
	if coverMode := testing.CoverMode(); coverMode != "" {
		testCmd = append(testCmd, "-covermode", coverMode, "-coverpkg", "./...")
	}
	if out, err := exec.Command("go", testCmd...).CombinedOutput(); err != nil {
		log.Fatalf("Error building lab test binary: %s (%s)", string(out), err)
	}

	originalWd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	// Make a copy of the testdata Git test project and chdir to it.
	repo := copyTestRepo(log.New(os.Stderr, "", log.LstdFlags))
	if err := os.Chdir(repo); err != nil {
		log.Fatalf("Error chdir to testdata: %s", err)
	}
	// Load config for non-testbinary based tests
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
		config["token"].(string))

	code := m.Run()

	if err := os.Chdir(originalWd); err != nil {
		log.Fatalf("Error chdir to original working dir: %s", err)
	}
	os.Remove(labBinaryPath)
	testdirs, err := filepath.Glob(os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata-*"))
	if err != nil {
		log.Printf("Error listing glob testdata-*: %s", err)
	}
	for _, dir := range testdirs {
		err := os.RemoveAll(dir)
		if err != nil {
			log.Printf("Error removing dir %s: %s", dir, err)
		}
	}

	os.Exit(code)
}

func TestRootCloneNoArg(t *testing.T) {
	cmd := exec.Command(labBinaryPath, "clone")
	b, _ := cmd.CombinedOutput()
	require.Contains(t, string(b), "You must specify a repository to clone.")
}

func TestRootGitCmd(t *testing.T) {
	cmd := exec.Command(labBinaryPath, "log", "-n", "1")
	b, _ := cmd.CombinedOutput()
	require.Contains(t, string(b), `commit 09b519cba018b707c98fc56e37df15806d89d866
Author: Zaq? Wiedmann <zaquestion@gmail.com>
Date:   Sun Apr 1 19:40:47 2018 -0700

    (ci) jobs with interleaved sleeps and prints`)
}

func TestRootNoArg(t *testing.T) {
	cmd := exec.Command(labBinaryPath)
	b, _ := cmd.CombinedOutput()
	assert.Contains(t, string(b), "usage: git [--version] [--help] [-C <path>]")
	assert.Contains(t, string(b), `These GitLab commands are provided by lab:

  ci            Work with GitLab CI pipelines and jobs`)
}

func TestRootVersion(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	RootCmd.Flag("version").Value.Set("true")
	RootCmd.Run(RootCmd, nil)

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	// back to normal state
	w.Close()
	os.Stdout = old // restoring the real stdout
	out := <-outC

	assert.Contains(t, out, "git version")
	assert.Contains(t, out, fmt.Sprintf("lab version %s", Version))
}

func TestGitHelp(t *testing.T) {
	cmd := exec.Command(labBinaryPath)
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	expected := string(b)
	expected = expected[:strings.LastIndex(strings.TrimSpace(expected), "\n")]

	tests := []struct {
		desc string
		Cmds []string
	}{
		{
			desc: "help arg",
			Cmds: []string{"help"},
		},
		{
			desc: "no arg",
			Cmds: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			cmd := exec.Command(labBinaryPath)
			if len(test.Cmds) >= 1 {
				cmd = exec.Command(labBinaryPath, test.Cmds...)
			}
			b, _ := cmd.CombinedOutput()
			res := string(b)
			res = res[:strings.LastIndex(strings.TrimSpace(res), "\n")]
			t.Log(expected)
			t.Log(res)
			assert.Equal(t, expected, res)
			assert.Contains(t, res, "usage: git [--version] [--help] [-C <path>]")
			assert.Contains(t, res, `These GitLab commands are provided by lab:

  ci            Work with GitLab CI pipelines and jobs`)
		})
	}
}

func Test_parseArgsStr(t *testing.T) {
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
			ExpectedString: "foo",
			ExpectedInt:    0,
			ExpectedErr:    "",
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
			ExpectedString: "asdf100",
			ExpectedInt:    0,
			ExpectedErr:    "",
		},
		{
			Name:           "2 arg str page",
			Args:           []string{"origin", "100"},
			ExpectedString: "origin",
			ExpectedInt:    100,
			ExpectedErr:    "",
		},
		{
			Name:           "2 arg valid str valid page",
			Args:           []string{"foo", "100"},
			ExpectedString: "foo",
			ExpectedInt:    100,
			ExpectedErr:    "",
		},
		{
			Name:           "2 arg valid str invalid page",
			Args:           []string{"foo", "asdf100"},
			ExpectedString: "foo",
			ExpectedInt:    0,
			ExpectedErr:    "strconv.ParseInt: parsing \"asdf100\": invalid syntax",
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			s, i, err := parseArgsStr(test.Args)
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
			Name:           "2 arg invalid remote invalid page",
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

func Test_parseArgsRemoteString(t *testing.T) {
	tests := []struct {
		Name           string
		Args           []string
		ExpectedRemote string
		ExpectedString string
		ExpectedErr    string
	}{
		{
			Name:           "No Args",
			Args:           nil,
			ExpectedRemote: "zaquestion/test",
			ExpectedString: "",
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg remote",
			Args:           []string{"lab-testing"},
			ExpectedRemote: "lab-testing/test",
			ExpectedString: "",
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg non remote",
			Args:           []string{"foo123"},
			ExpectedRemote: "zaquestion/test",
			ExpectedString: "foo123",
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg page",
			Args:           []string{"100"},
			ExpectedRemote: "zaquestion/test",
			ExpectedString: "100",
			ExpectedErr:    "",
		},
		{
			Name:           "2 arg remote and string",
			Args:           []string{"origin", "foo123"},
			ExpectedRemote: "zaquestion/test",
			ExpectedString: "foo123",
			ExpectedErr:    "",
		},
		{
			Name:           "2 arg invalid remote and string",
			Args:           []string{"foo", "string123"},
			ExpectedRemote: "",
			ExpectedString: "",
			ExpectedErr:    "foo is not a valid remote",
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			r, s, err := parseArgsRemoteString(test.Args)
			if err != nil {
				assert.EqualError(t, err, test.ExpectedErr)
			}
			assert.Equal(t, test.ExpectedRemote, r)
			assert.Equal(t, test.ExpectedString, s)
		})
	}
}

type fatalLogger interface {
	Fatal(...interface{})
}

// copyTestRepo creates a copy of the testdata directory (contains a Git repo) in
// the project root with a random dir name. It returns the absolute path of the
// new testdata dir.
// Note: testdata-* must be in the .gitignore or the copies will create write
// errors as Git attempts to add the Git repo to the the project repo's index.
func copyTestRepo(log fatalLogger) string {
	dst, err := filepath.Abs(os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata-" + strconv.Itoa(int(rand.Uint64()))))
	if err != nil {
		log.Fatal(err)
	}
	src, err := filepath.Abs(os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata"))
	if err != nil {
		log.Fatal(err)
	}
	if err := copy.Copy(src, dst); err != nil {
		log.Fatal(err)
	}
	// Move the test.git dir into the expected path at .git
	if err := os.Rename(dst+"/test.git", dst+"/.git"); err != nil {
		log.Fatal(err)
	}
	return dst
}

// getAppOutput splits and truncates the list of strings returned from the "lab"
// test binary to remove the test-specific output. It use "PASS" as a marker for
// the end of the app output and the beginning of the test output.
func getAppOutput(output []byte) []string {
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if line == "PASS" {
			return lines[:i]
		}
	}
	return lines
}
