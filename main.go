package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/rsteube/carapace"
	"github.com/spf13/viper"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/cmd"
	"github.com/zaquestion/lab/internal/config"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// version gets set on releases during build by goreleaser.
var version = "master"

func loadConfig() (string, string, string, bool) {

	// Attempt to auto-configure for GitLab CI.
	// Always do this before reading in the config file o/w CI will end up
	// with the wrong data.
	host, user, token := config.CI()
	if host != "" && user != "" && token != "" {
		return host, user, token, false
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
	labconfpath := confpath+"/lab"
	if _, err := os.Stat(confpath); os.IsNotExist(err) {
		os.MkdirAll(confpath, 0700)
	}

	// Convert old hcl files to toml format.
	// NO NEW FILES SHOULD BE ADDED BELOW.
	config.ConvertHCLtoTOML(".", ".", "lab")
	config.ConvertHCLtoTOML(confpath, labconfpath, "lab")
	var labgitDir string
	gitDir, err := git.GitDir()
	if err == nil {
		labgitDir = gitDir+"/lab"
		config.ConvertHCLtoTOML(gitDir, labgitDir, "lab")
		config.ConvertHCLtoTOML(labgitDir, labgitDir, "show_metadata")
	}

	viper.SetConfigName("lab")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath(labconfpath)
	if labgitDir != "" {
		viper.AddConfigPath(labgitDir)
	}

	viper.SetEnvPrefix("LAB")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if _, ok := viper.ReadInConfig().(viper.ConfigFileNotFoundError); ok {
		err := config.New(path.Join(labconfpath, "lab.toml"), os.Stdin)
		if err != nil {
			log.Fatal(err)
		}

		err = viper.ReadInConfig()
		if err != nil {
			log.Fatal(err)
		}
	}

	host = viper.GetString("core.host")
	user = viper.GetString("core.user")
	token = viper.GetString("core.token")
	tlsSkipVerify := viper.GetBool("tls.skip_verify")

	if host != "" && user != "" && token != "" {
		return host, user, token, tlsSkipVerify
	}

	user = getUser(host, token, tlsSkipVerify)
	if strings.TrimSpace(os.Getenv("LAB_CORE_TOKEN")) == "" && strings.TrimSpace(os.Getenv("LAB_CORE_HOST")) == "" {
		viper.Set("core.user", user)
		viper.WriteConfig()
	}

	return host, user, token, tlsSkipVerify
}

func loadTLSCerts() string {
	c := viper.AllSettings()

	var tls map[string]interface{}
	switch v := c["tls"].(type) {
	// Most run this is the type
	case []map[string]interface{}:
		tls = v[0]
	// On the first run when the cfg is created it comes in as this type
	// for whatever reason
	case map[string]interface{}:
		tls = v
	}

	for _, v := range []string{"ca_file"} {
		if _, ok := tls[v]; !ok {
			return ""
		}
	}

	if v := viper.GetString("tls.ca_file"); v != "" {
		tls["ca_file"] = v
	}

	return tls["ca_file"].(string)
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

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	cmd.Version = version
	if !skipInit() {
		ca := loadTLSCerts()
		h, u, t, skipVerify := loadConfig()

		if ca != "" {
			lab.InitWithCustomCA(h, u, t, ca)
		} else {
			lab.Init(h, u, t, skipVerify)
		}
	}
	cmd.Execute()
}

func skipInit() bool {
	if len(os.Args) <= 1 {
		return false
	}
	switch os.Args[1] {
	case "completion":
		return true
	case "_carapace":
		return !carapace.IsCallback()
	default:
		return false
	}
}
