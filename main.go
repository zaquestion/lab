package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"syscall"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	gitconfig "github.com/tcnksm/go-gitconfig"
	"github.com/zaquestion/lab/cmd"
	lab "github.com/zaquestion/lab/internal/gitlab"
	"golang.org/x/crypto/ssh/terminal"
)

// version gets set on releases during build by goreleaser.
var version = "master"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	cmd.Version = version

	home, err := homedir.Dir()
	if err != nil {
		log.Fatal(err)
	}
	confpath := path.Join(home, ".config")
	if _, err := os.Stat(confpath); os.IsNotExist(err) {
		os.Mkdir(confpath, 0700)
	}

	viper.SetConfigName("lab")
	viper.SetConfigType("hcl")
	viper.AddConfigPath(".")
	viper.AddConfigPath(confpath)
	viper.AutomaticEnv()
	err = viper.ReadInConfig()
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		host, user, token := legacyLoadConfig()
		writeConfig(confpath, host, user, token)
		err = viper.ReadInConfig()
		if err != nil {
			log.Fatal(err)
		}
	}

	c := viper.AllSettings()["core"]
	var config map[string]interface{}
	switch v := c.(type) {
	// Most run this is the type
	case []map[string]interface{}:
		config = v[0]
	// On the first run when the config is created it comes in as this type
	// for whatever reason
	case map[string]interface{}:
		config = v
	}

	lab.Init(
		config["host"].(string),
		config["user"].(string),
		config["token"].(string))

	cmd.Execute()
}

func writeConfig(confpath, host, user, token string) {
	viper.Set("core.host", host)
	viper.Set("core.user", user)
	viper.Set("core.token", token)
	err := viper.WriteConfigAs(path.Join(confpath, "lab.hcl"))
	if err != nil {
		log.Fatal(err)
	}
}

const defaultGitLabHost = "https://gitlab.com"

// legacyLoadConfig handles all of the credential setup and prompts for user
// input when not present
func legacyLoadConfig() (host, user, token string) {
	reader := bufio.NewReader(os.Stdin)
	var err error
	host, err = gitconfig.Entire("gitlab.host")
	if err != nil {
		fmt.Printf("Enter default GitLab host (default: %s): ", defaultGitLabHost)
		host, err = reader.ReadString('\n')
		host = strings.TrimSpace(host)
		if err != nil {
			log.Fatal(err)
		}
		if host == "" {
			host = defaultGitLabHost
		}
	}
	var errt error
	user, err = gitconfig.Entire("gitlab.user")
	token, errt = gitconfig.Entire("gitlab.token")
	if err != nil {
		fmt.Print("Enter default GitLab user: ")
		user, err = reader.ReadString('\n')
		user = strings.TrimSpace(user)
		if err != nil {
			log.Fatal(err)
		}
		if user == "" {
			log.Fatal("git config gitlab.user must be set")
		}
		tokenURL := path.Join(host, "profile/personal_access_tokens")

		// If the default user is being set this is the first time lab
		// is being run.
		if errt != nil {
			fmt.Printf("Create a token here: %s\nEnter default GitLab token (scope: api): ", tokenURL)
			byteToken, err := terminal.ReadPassword(int(syscall.Stdin))
			if err != nil {
				log.Fatal(err)
			}
			token = strings.TrimSpace(string(byteToken))
		}
	}
	return
}
