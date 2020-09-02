package config

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
)

const defaultGitLabHost = "https://gitlab.com"

// New prompts the user for the default config values to use with lab, and save
// them to the provided confpath (default: ~/.config/lab.hcl)
func New(confpath string, r io.Reader) error {
	var (
		reader      = bufio.NewReader(r)
		host, token string
		err         error
	)
	// If core host is set in the environment (LAB_CORE_HOST) we only want
	// to prompt for the token. We'll use the environments host and place
	// it in the config. In the event both the host and token are in the
	// env, this function shouldn't be called in the first place
	if viper.GetString("core.host") == "" {
		fmt.Printf("Enter GitLab host (default: %s): ", defaultGitLabHost)
		host, err = reader.ReadString('\n')
		host = strings.TrimSpace(host)
		if err != nil {
			return err
		}
		if host == "" {
			host = defaultGitLabHost
		}
	} else {
		// Required to correctly write config
		host = viper.GetString("core.host")
	}

	tokenURL, err := url.Parse(host)
	if err != nil {
		return err
	}
	tokenURL.Path = "profile/personal_access_tokens"

	fmt.Printf("Create a token here: %s\nEnter default GitLab token (scope: api): ", tokenURL.String())
	token, err = readPassword()
	if err != nil {
		return err
	}

	viper.Set("core.host", host)
	viper.Set("core.token", token)
	if err := viper.WriteConfigAs(confpath); err != nil {
		return err
	}
	fmt.Printf("\nConfig saved to %s\n", confpath)
	return nil
}

var readPassword = func() (string, error) {
	byteToken, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(byteToken)), nil
}

// CI returns credentials suitable for use within GitLab CI or empty strings if
// none found.
func CI() (string, string, string) {
	ciToken := os.Getenv("CI_JOB_TOKEN")
	if ciToken == "" {
		return "", "", ""
	}
	ciHost := strings.TrimSuffix(os.Getenv("CI_PROJECT_URL"), os.Getenv("CI_PROJECT_PATH"))
	if ciHost == "" {
		return "", "", ""
	}
	ciUser := os.Getenv("GITLAB_USER_LOGIN")

	return ciHost, ciUser, ciToken
}

// ConvertHCLtoTOML() converts an .hcl file to a .toml file
func ConvertHCLtoTOML(oldpath string, newpath string, file string) {
	oldconfig := oldpath + "/" + file + ".hcl"
	newconfig := newpath + "/" + file + ".toml"

	_, err := os.Stat(oldconfig)
	if os.IsNotExist(err) {
		fmt.Println("oldfile not found", oldconfig)
		return
	}

	_, err = os.Stat(newconfig)
	if err == nil {
		fmt.Println("newfile found", newconfig)
		return
	}

	// read in the old config HCL file and write out the new TOML file
	viper.Reset()
	viper.SetConfigName("lab")
	viper.SetConfigType("hcl")
	viper.AddConfigPath(oldpath)
	viper.ReadInConfig()
	viper.SetConfigType("toml")
	viper.WriteConfigAs(newconfig)

	// delete the old config HCL file
	err = os.Remove(oldconfig)
	if err != nil {
		fmt.Println("Warning: Could not delete old config file", oldconfig)
	}

	// HACK
	// viper HCL parsing is broken and simply translating it to a TOML file
	// results in a broken toml file.  The issue is that there are double
	// square brackets for each entry where there should be single
	// brackets.  Note: this hack only works because the config file is
	// simple and doesn't contain deeply embedded config entries.
	text, err := ioutil.ReadFile(newconfig)
	if err != nil {
		log.Fatal(err)
	}

	text = bytes.Replace(text, []byte("[["), []byte("["), -1)
	text = bytes.Replace(text, []byte("]]"), []byte("]"), -1)

	if err = ioutil.WriteFile(newconfig, text, 0666); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// END HACK

	fmt.Println("INFO: Converted old config", oldconfig, "to new config", newconfig)
}
