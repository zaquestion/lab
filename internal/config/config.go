package config

import (
	"bufio"
	"fmt"
	"io"
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
