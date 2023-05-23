package config

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initMainConfig(testconf string) {
	MainConfig = viper.New()
	MainConfig.SetConfigName("lab")
	MainConfig.SetConfigType("toml")
	MainConfig.AddConfigPath(testconf)
}

func resetMainConfig() {
	// *viper.Viper.Reset() does not exist so just set MainConfig to nil for testing
	MainConfig = nil
}

func TestNewConfig(t *testing.T) {
	testconf := t.TempDir()

	t.Run("create config", func(t *testing.T) {
		old := os.Stdout // keep backup of the real stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		var buf bytes.Buffer
		fmt.Fprintln(&buf, "https://gitlab.zaquestion.io")

		oldreadPassword := readPassword
		readPassword = func(bufio.Reader) (string, string, error) {
			return "abcde12345", "", nil
		}
		defer func() {
			readPassword = oldreadPassword
		}()

		initMainConfig(testconf)
		err := New(testconf, &buf)
		if err != nil {
			t.Fatal(err)
		}

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

		assert.Contains(t, out, "Enter GitLab host (default: https://gitlab.com): ")

		cfg, err := os.Open(path.Join(testconf, "lab.toml"))
		if err != nil {
			t.Fatal(err)
		}

		cfgData, err := ioutil.ReadAll(cfg)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, `[core]
host = 'https://gitlab.zaquestion.io'
token = 'abcde12345'
`, string(cfgData))
	})
	resetMainConfig()
}

func TestNewConfigHostOverride(t *testing.T) {
	testconf := t.TempDir()

	os.Setenv("LAB_CORE_HOST", "https://gitlab2.zaquestion.io")

	t.Run("create config", func(t *testing.T) {

		initMainConfig(testconf)
		MainConfig.SetEnvPrefix("LAB")
		MainConfig.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		MainConfig.AutomaticEnv()

		require.Equal(t, "https://gitlab2.zaquestion.io", MainConfig.GetString("core.host"))

		old := os.Stdout // keep backup of the real stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		oldreadPassword := readPassword
		readPassword = func(bufio.Reader) (string, string, error) {
			return "abcde12345", "", nil
		}
		defer func() {
			readPassword = oldreadPassword
		}()

		var buf bytes.Buffer

		err := New(testconf, &buf)
		if err != nil {
			t.Fatal(err)
		}

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

		assert.NotContains(t, out, "Enter GitLab host")

		cfg, err := os.Open(path.Join(testconf, "lab.toml"))
		if err != nil {
			t.Fatal(err)
		}

		cfgData, err := ioutil.ReadAll(cfg)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, `[core]
host = 'https://gitlab2.zaquestion.io'
token = 'abcde12345'
`, string(cfgData))
	})
	resetMainConfig()
}

func TestNewLoadTokenConfig(t *testing.T) {
	testconf := t.TempDir()

	t.Run("create load_token config", func(t *testing.T) {
		old := os.Stdout // keep backup of the real stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		var buf bytes.Buffer
		fmt.Fprintln(&buf, "https://gitlab.zaquestion.io")

		oldreadPassword := readPassword
		readPassword = func(bufio.Reader) (string, string, error) {
			return "", "bash echo abcde12345", nil
		}
		defer func() {
			readPassword = oldreadPassword
		}()

		initMainConfig(testconf)
		err := New(testconf, &buf)
		if err != nil {
			t.Fatal(err)
		}

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

		assert.Contains(t, out, "Enter GitLab host (default: https://gitlab.com): ")

		cfg, err := os.Open(path.Join(testconf, "lab.toml"))
		if err != nil {
			t.Fatal(err)
		}

		cfgData, err := ioutil.ReadAll(cfg)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, `[core]
host = 'https://gitlab.zaquestion.io'
load_token = 'bash echo abcde12345'
`, string(cfgData))
	})
	resetMainConfig()
}

func TestConvertHCLtoTOML(t *testing.T) {
	tmpDir := t.TempDir()
	oldCnfPath := filepath.Join(tmpDir, "lab.hcl")
	newCnfPath := filepath.Join(tmpDir, "lab.toml")
	oldCnf, err := os.Create(oldCnfPath)
	if err != nil {
		t.Fatal(err)
	}
	oldCnf.WriteString(`"core" = {
  "host" = "https://gitlab.com"
  "token" = "foobar"
  "user" = "lab-testing"
}`)

	ConvertHCLtoTOML(tmpDir, tmpDir, "lab")

	_, err = os.Stat(oldCnfPath)
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(newCnfPath)
	assert.NoError(t, err)

	newCnf, err := os.Open(newCnfPath)
	if err != nil {
		t.Fatal(err)
	}
	cfgData, err := ioutil.ReadAll(newCnf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, `[core]
host = 'https://gitlab.com'
token = 'foobar'
user = 'lab-testing'
`, string(cfgData))
}

func TestTokenTest(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "lab.toml")
	config, err := os.Create(configPath)
	if err != nil {
		t.Fatal(err)
	}
	config.WriteString(`
[core]
  host = "https://gitlab.com"
  token = "foobar"
  user = "lab-testing"
`)

	initMainConfig(tmpDir)
	MainConfig.ReadInConfig()
	token := GetToken()
	os.Remove(configPath + "/lab.toml")
	resetMainConfig()
	assert.Equal(t, "foobar", token)
}

func TestLoadTokenTest(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "lab.toml")
	config, err := os.Create(configPath)
	if err != nil {
		t.Fatal(err)
	}
	config.WriteString(`
[core]
  host = "https://gitlab.com"
  load_token = "echo foobar"
  user = "lab-testing"
`)

	initMainConfig(tmpDir)
	MainConfig.ReadInConfig()
	token := GetToken()
	os.Remove(configPath + "/lab.toml")
	resetMainConfig()
	assert.Equal(t, "foobar", token)
}
