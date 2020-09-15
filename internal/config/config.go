package config

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"

	"github.com/spf13/viper"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	"golang.org/x/crypto/ssh/terminal"
)

const defaultGitLabHost = "https://gitlab.com"

var (
	MainConfig *viper.Viper
)

// New prompts the user for the default config values to use with lab, and save
// them to the provided confpath (default: ~/.config/lab.hcl)
func New(confpath string, r io.Reader) error {
	var (
		reader                 = bufio.NewReader(r)
		host, token, loadToken string
		err                    error
	)

	confpath = path.Join(confpath, "lab.toml")
	// If core host is set in the environment (LAB_CORE_HOST) we only want
	// to prompt for the token. We'll use the environments host and place
	// it in the config. In the event both the host and token are in the
	// env, this function shouldn't be called in the first place
	if MainConfig.GetString("core.host") == "" {
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
		host = MainConfig.GetString("core.host")
	}

	MainConfig.Set("core.host", host)

	token, loadToken, err = readPassword(*reader)
	if err != nil {
		return err
	}
	if token != "" {
		MainConfig.Set("core.token", token)
	} else if loadToken != "" {
		MainConfig.Set("core.load_token", loadToken)
	}

	if err := MainConfig.WriteConfigAs(confpath); err != nil {
		return err
	}
	fmt.Printf("\nConfig saved to %s\n", confpath)
	return nil
}

var readPassword = func(reader bufio.Reader) (string, string, error) {
	var loadToken string

	tokenURL, err := url.Parse(viper.GetString("core.host"))
	if err != nil {
		return "", "", err
	}
	tokenURL.Path = "profile/personal_access_tokens"

	fmt.Printf("Create a token here: %s\nEnter default GitLab token (scope: api), or leave blank to provide a command to load the token: ", tokenURL.String())
	byteToken, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(string(byteToken)) == "" {
		fmt.Printf("\nEnter command to load the token:")
		loadToken, err = reader.ReadString('\n')
		if err != nil {
			return "", "", err
		}
	}

	if strings.TrimSpace(string(byteToken)) == "" && strings.TrimSpace(loadToken) == "" {
		log.Fatal("Error: No token provided.  A token can be created at ", tokenURL.String())
	}
	return strings.TrimSpace(string(byteToken)), strings.TrimSpace(loadToken), nil
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

	if _, err := os.Stat(oldconfig); os.IsNotExist(err) {
		return
	}

	if _, err := os.Stat(newconfig); err == nil {
		return
	}

	// read in the old config HCL file and write out the new TOML file
	oldConfig := viper.New()
	oldConfig.SetConfigName("lab")
	oldConfig.SetConfigType("hcl")
	oldConfig.AddConfigPath(oldpath)
	oldConfig.ReadInConfig()
	oldConfig.SetConfigType("toml")
	oldConfig.WriteConfigAs(newconfig)

	// delete the old config HCL file
	if err := os.Remove(oldconfig); err != nil {
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

func getUser(host, token string, skipVerify bool) string {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: skipVerify,
			},
		},
	}
	lab, _ := gitlab.NewClient(token, gitlab.WithHTTPClient(httpClient), gitlab.WithBaseURL(host+"/api/v4"))
	u, _, err := lab.Users.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	return u.Username
}

// GetToken returns a token string from the config file.
// The token string can be cleartext or returned from a password manager or
// encryption utility.
func GetToken() string {
	token := MainConfig.GetString("core.token")
	if token == "" && MainConfig.GetString("core.load_token") != "" {
		// args[0] isn't really an arg ;)
		args := strings.Split(MainConfig.GetString("core.load_token"), " ")
		_token, err := exec.Command(args[0], args[1:]...).Output()
		if err != nil {
			log.Fatal(err)
		}
		token = string(_token)
		// tools like pass and a simple bash script add a '\n' to
		// their output which confuses the gitlab WebAPI
		if token[len(token)-1:] == "\n" {
			token = strings.TrimSuffix(token, "\n")
		}
	}
	return token
}

// LoadMainConfig() loads the main config file and returns a tuple of
//  host, user, token, ca_file, skipVerify
func LoadMainConfig() (string, string, string, string, bool) {

	// Attempt to auto-configure for GitLab CI.
	// Always do this before reading in the config file o/w CI will end up
	// with the wrong data.
	host, user, token := CI()
	if host != "" && user != "" && token != "" {
		return host, user, token, "", false
	}

	// Try to find XDG_CONFIG_HOME which is declared in XDG base directory
	// specification and use it's location as the config directory
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	confpath := os.Getenv("XDG_CONFIG_HOME")
	if confpath == "" {
		confpath = path.Join(home, ".config")
	}
	labconfpath := confpath + "/lab"
	if _, err := os.Stat(labconfpath); os.IsNotExist(err) {
		os.MkdirAll(labconfpath, 0700)
	}

	// Convert old hcl files to toml format.
	// NO NEW FILES SHOULD BE ADDED BELOW.
	ConvertHCLtoTOML(confpath, labconfpath, "lab")
	ConvertHCLtoTOML(".", ".", "lab")
	var labgitDir string
	gitDir, err := git.GitDir()
	if err == nil {
		labgitDir = gitDir + "/lab"
		ConvertHCLtoTOML(gitDir, labgitDir, "lab")
		ConvertHCLtoTOML(labgitDir, labgitDir, "show_metadata")
	}

	MainConfig = viper.New()
	MainConfig.SetConfigName("lab")
	MainConfig.SetConfigType("toml")
	MainConfig.AddConfigPath(".")
	MainConfig.AddConfigPath(labconfpath)
	if labgitDir != "" {
		MainConfig.AddConfigPath(labgitDir)
	}

	MainConfig.SetEnvPrefix("LAB")
	MainConfig.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	MainConfig.AutomaticEnv()

	if _, ok := MainConfig.ReadInConfig().(viper.ConfigFileNotFoundError); ok {
		err := New(labconfpath, os.Stdin)
		if err != nil {
			log.Fatal(err)
		}

		err = MainConfig.ReadInConfig()
		if err != nil {
			log.Fatal(err)
		}
	}

	host = MainConfig.GetString("core.host")
	user = MainConfig.GetString("core.user")
	token = GetToken()
	tlsSkipVerify := MainConfig.GetBool("tls.skip_verify")
	ca_file := MainConfig.GetString("tls.ca_file")

	if user == "" {
		user = getUser(host, token, tlsSkipVerify)
		if strings.TrimSpace(os.Getenv("LAB_CORE_TOKEN")) == "" && strings.TrimSpace(os.Getenv("LAB_CORE_HOST")) == "" {
			MainConfig.Set("core.user", user)
			MainConfig.WriteConfig()
		}
	}

	return host, user, token, ca_file, tlsSkipVerify
}

// default path for work tree config file
var worktreepath = ".git/lab/"

// LoadWorkTreeConfig opens and reads the .git/lab/[cmd]_string.toml
// metadata file
func LoadWorkTreeConfig(cmd string) {
	viper.Reset()
	viper.AddConfigPath(worktreepath)
	viper.SetConfigName(cmd + "_metadata")
	viper.SetConfigType("toml")

	if _, ok := viper.ReadInConfig().(viper.ConfigFileNotFoundError); ok {
		if _, err := os.Stat(worktreepath); os.IsNotExist(err) {
			os.MkdirAll(worktreepath, os.ModePerm)
		}
		if err := viper.WriteConfigAs(worktreepath + cmd + "_metadata.toml"); err != nil {
			log.Fatal(err)
		}
		if err := viper.ReadInConfig(); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := viper.ReadInConfig(); err != nil {
			log.Fatal(err)
		}
	}
}

// WriteWorkTreeConfig saves the .git/lab/[cmd]_string.toml metadata file
func WriteWorkTreeConfig(cmd string) {
	viper.WriteConfigAs(worktreepath + cmd + "_metadata.toml")
}

// FinishWorkTreeConfig closes the .git/lab/[cmd]_string.toml metadata file
func FinishWorkTreeConfig() {
	viper.Reset()
}
