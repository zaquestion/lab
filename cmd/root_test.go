package cmd

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/acarl005/stripansi"
	"github.com/otiai10/copy"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
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
	repo := copyTestRepo(log)
	if err := os.Chdir(repo); err != nil {
		log.Fatalf("Error chdir to testdata: %s", err)
	}
	// Load config for non-testbinary based tests
	viper.SetConfigName("lab")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	err = viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
	host := viper.GetString("core.host")
	token := viper.GetString("core.token")

	client, _ := gitlab.NewClient(token, gitlab.WithBaseURL(host+"/api/v4"))
	u, _, err := client.Users.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	lab.Init(context.Background(), host, u.Username, token, false)

	// Make "origin" the default remote for test cases calling
	// cmd.Run() directly, instead of launching the labBinaryPath
	// for getting these vars correctly set through Execute().
	defaultRemote = "origin"
	forkRemote = "origin"
	code := m.Run()

	if err := os.Chdir(originalWd); err != nil {
		log.Fatalf("Error chdir to original working dir: %s", err)
	}
	os.Remove(labBinaryPath)
	testdirs, err := filepath.Glob(os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata-*"))
	if err != nil {
		log.Infof("Error listing glob testdata-*: %s", err)
	}
	for _, dir := range testdirs {
		err := os.RemoveAll(dir)
		if err != nil {
			log.Infof("Error removing dir %s: %s", dir, err)
		}
	}

	os.Exit(code)
}

func TestRootCloneNoArg(t *testing.T) {
	cmd := exec.Command(labBinaryPath, "clone")
	b, _ := cmd.CombinedOutput()
	require.Contains(t, string(b), "You must specify a repository to clone.")
}

func TestRootNoArg(t *testing.T) {
	cmd := exec.Command(labBinaryPath)
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	assert.Contains(t, string(b), `lab: A GitLab Command Line Interface Utility

Usage:
  lab [flags]
  lab [command]`)
}

func TestRootHelp(t *testing.T) {
	cmd := exec.Command(labBinaryPath, "help")
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	res := string(b)
	assert.Contains(t, res, `Show the help for lab

Usage:
  lab help [command [subcommand...]] [flags]`)
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
	dstDir := strconv.FormatUint(rand.Uint64(), 10)
	dst, err := filepath.Abs(
		os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata-" + dstDir))
	if err != nil {
		log.Fatal(err)
	}
	src, err := filepath.Abs(
		os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata"))
	if err != nil {
		log.Fatal(err)
	}
	if err := copy.Copy(src, dst); err != nil {
		log.Fatal(err)
	}
	// Move the test.git dir into the expected path at .git
	if err := copy.Copy(dst+"/test.git", dst+"/.git"); err != nil {
		log.Fatal(err)
	}
	return dst
}

func whoAmI() string {
	cmd := exec.Command("whoami")
	whoami, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	return string(whoami)
}

func configFile() string {
	str := "/home/" + whoAmI() + "/.config/lab"
	str = strings.Replace(str, "\n", "", -1)
	if _, err := os.Stat(str); os.IsNotExist(err) {
		os.MkdirAll(str, os.ModePerm)
	}
	str = str + "/lab.toml"
	return str
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

func setConfigValues(repo string, configVal string, gitVal string) error {
	err := copy.Copy(repo+"/lab.toml", configFile())
	if err != nil {
		log.Errorln(err)
		return err
	}

	configfile, err := os.OpenFile(configFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := configfile.WriteString("\n[mr_show]\n  comments = " + configVal + "\n"); err != nil {
		log.Fatal(err)
	}
	configfile.Close()

	err = os.Mkdir(repo+"/.git/lab/", 0700)
	if err != nil {
		log.Fatal(err)
	}
	gitfile, err := os.OpenFile(repo+"/.git/lab/lab.toml", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	if _, err = gitfile.WriteString("\n[mr_show]\n  comments = " + gitVal + "\n"); err != nil {
		log.Fatal(err)
	}
	gitfile.Close()

	return nil
}

// There isn't a really good way to test the config override
// infrastruture, so just call 'mr show' and set 'mr_show.comments'
func Test_config_gitConfig_FF(t *testing.T) {
	repo := copyTestRepo(t)

	err := setConfigValues(repo, "false", "false")
	if err != nil {
		t.Skip(err)
	}
	os.Remove(repo + "/lab.toml")

	cmd := exec.Command(labBinaryPath, "mr", "show", "1")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}
	out := stripansi.Strip(string(b))

	os.Remove(configFile())
	// both configs set to false, comments should not be output
	require.NotContains(t, string(out), `commented at`)
}

func Test_config_gitConfig_FT(t *testing.T) {
	repo := copyTestRepo(t)

	err := setConfigValues(repo, "false", "true")
	if err != nil {
		t.Skip(err)
	}
	os.Remove(repo + "/lab.toml")

	cmd := exec.Command(labBinaryPath, "mr", "show", "1")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}
	out := stripansi.Strip(string(b))

	os.Remove(configFile())
	// .config set to false and .git set to true, comments should be
	// output
	require.Contains(t, string(out), `commented at`)
}

func Test_config_gitConfig_TF(t *testing.T) {
	repo := copyTestRepo(t)

	err := setConfigValues(repo, "true", "false")
	if err != nil {
		t.Skip(err)
	}
	os.Remove(repo + "/lab.toml")

	cmd := exec.Command(labBinaryPath, "mr", "show", "1")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}
	out := stripansi.Strip(string(b))

	os.Remove(configFile())
	// .config set to true and .git set to false, comments should not be
	// output
	require.NotContains(t, string(out), `commented at`)
}

func Test_config_gitConfig_TT(t *testing.T) {
	repo := copyTestRepo(t)

	err := setConfigValues(repo, "true", "true")
	if err != nil {
		t.Skip(err)
	}
	os.Remove(repo + "/lab.toml")

	cmd := exec.Command(labBinaryPath, "mr", "show", "1")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}
	out := stripansi.Strip(string(b))

	os.Remove(configFile())
	// both configs set to true, comments should be output
	require.Contains(t, string(out), `commented at`)
}

// Some flag and config tests do not have to be run.
// flag not set, config true == comments
//   This case is handled by Test_config_gitConfig_TT
// flag not set, config false == no comments
//   This case is handled by Test_config_gitConfig_FF
// flag not set, config not set == no comments
// flag set, config not set == comments
//   These case are handled in cmd/mr_show_test.go

// flag set, config true == comments
func Test_flag_config_TT(t *testing.T) {
	repo := copyTestRepo(t)

	err := setConfigValues(repo, "true", "true")
	if err != nil {
		t.Skip(err)
	}
	os.Remove(repo + "/lab.toml")

	cmd := exec.Command(labBinaryPath, "mr", "show", "1", "--comments")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}
	out := stripansi.Strip(string(b))

	os.Remove(configFile())
	// both configs set to true, comments should be output
	require.Contains(t, string(out), `commented at`)
}

// flag set, config false == comments
func Test_flag_config_TF(t *testing.T) {
	repo := copyTestRepo(t)

	err := setConfigValues(repo, "false", "false")
	if err != nil {
		t.Skip(err)
	}
	os.Remove(repo + "/lab.toml")

	cmd := exec.Command(labBinaryPath, "mr", "show", "1", "--comments")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}
	out := stripansi.Strip(string(b))

	os.Remove(configFile())
	// both configs set to true, comments should be output
	require.Contains(t, string(out), `commented at`)
}

// flag (explicitly) unset, config true == no comments
func Test_flag_config_FT(t *testing.T) {
	repo := copyTestRepo(t)

	err := setConfigValues(repo, "true", "true")
	if err != nil {
		t.Skip(err)
	}
	os.Remove(repo + "/lab.toml")

	cmd := exec.Command(labBinaryPath, "mr", "show", "1", "--comments=false")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Error(err)
	}
	out := stripansi.Strip(string(b))

	os.Remove(configFile())
	// configs overridden on the command line, comments should not be output
	require.NotContains(t, string(out), `commented at`)
}

// Make sure the version command don't break things in the future
func Test_versionCmd(t *testing.T) {

	t.Run("version", func(t *testing.T) {
		labCmd := exec.Command(labBinaryPath, "version")
		out, err := labCmd.CombinedOutput()
		if err != nil {
			t.Log(string(out))
			t.Fatal(err)
		}
		assert.Contains(t, string(out), "lab version "+Version)
	})
	t.Run("--version", func(t *testing.T) {
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

		assert.Contains(t, out, "lab version "+Version)
	})
}
