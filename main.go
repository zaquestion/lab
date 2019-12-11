package main

import (
	"log"
	"os"
	"path"
	"strings"

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
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// Try XDG_CONFIG_HOME which is declared in XDG base directory specification
	confpath := os.Getenv("XDG_CONFIG_HOME")
	if confpath == "" {
		confpath = path.Join(home, ".config")
	}
	if _, err := os.Stat(confpath); os.IsNotExist(err) {
		os.Mkdir(confpath, 0700)
	}

	viper.SetConfigName("lab")
	viper.SetConfigType("hcl")
	viper.AddConfigPath(".")
	viper.AddConfigPath(confpath)
	gitDir, err := git.GitDir()
	if err == nil {
		viper.AddConfigPath(gitDir)
	}

	viper.SetEnvPrefix("LAB")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	var tlsSkipVerify bool
	tlsSkipVerify = viper.GetBool("tls.skip_verify")

	host, user, token := viper.GetString("core.host"), viper.GetString("core.user"), viper.GetString("core.token")
	if host != "" && user != "" && token != "" {
		return host, user, token, tlsSkipVerify
	} else if host != "" && token != "" {
		user = getUser(host, token)
		return host, user, token, tlsSkipVerify
	}

	// Attempt to auto-configure for GitLab CI
	host, user, token = config.CI()
	if host != "" && user != "" && token != "" {
		return host, user, token, tlsSkipVerify
	}

	if _, ok := viper.ReadInConfig().(viper.ConfigFileNotFoundError); ok {
		if err := config.New(path.Join(confpath, "lab.hcl"), os.Stdin); err != nil {
			log.Fatal(err)
		}

		if err := viper.ReadInConfig(); err != nil {
			log.Fatal(err)
		}
	}

	c := viper.AllSettings()
	var cfg map[string]interface{}
	switch v := c["core"].(type) {
	// Most run this is the type
	case []map[string]interface{}:
		cfg = v[0]
	// On the first run when the cfg is created it comes in as this type
	// for whatever reason
	case map[string]interface{}:
		cfg = v
	}

	for _, v := range []string{"host", "token"} {
		if cv, ok := cfg[v]; !ok {
			log.Println(cv)
			log.Fatalf("missing config value core.%s in %s", v, viper.ConfigFileUsed())
		}
	}

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
	if v, ok := tls["skip_verify"]; ok {
		tlsSkipVerify = v.(bool)
	}

	// Set environment overrides
	// Note: the code below that uses `cfg["host"]` to access these values
	// is tough to simplify since cfg["host"] is accessing the array "core"
	// and viper.GetString("core.host") is expecting a non-array so it
	// doens't match
	if v := viper.GetString("core.host"); v != "" {
		cfg["host"] = v
	}
	if v := viper.GetString("core.token"); v != "" {
		cfg["token"] = v
	}
	if v := viper.Get("tls.skip_verify"); v != nil {
		tlsSkipVerify = v.(string) == "true"
	}
	host = cfg["host"].(string)
	token = cfg["token"].(string)
	user = getUser(host, token)
	viper.Set("core.user", user)
	return host, user, token, tlsSkipVerify
}

func loadTLSCerts() (string, string, string) {
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

	for _, v := range []string{"ca_file", "cert_file", "key_file"} {
		if _, ok := tls[v]; !ok {
			return "", "", ""
		}
	}

	if v := viper.GetString("tls.ca_file"); v != "" {
		tls["ca_file"] = v
	}
	if v := viper.GetString("tls.cert_file"); v != "" {
		tls["cert_file"] = v
	}
	if v := viper.GetString("tls.ca_file"); v != "" {
		tls["key_file"] = v
	}

	return tls["ca_file"].(string), tls["cert_file"].(string), tls["key_file"].(string)
}

func getUser(host, token string) string {
	lab := gitlab.NewClient(nil, token)
	lab.SetBaseURL(host + "/api/v4")
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
		ca, cert, key := loadTLSCerts()
		h, u, t, skipVerify := loadConfig()

		if ca != "" && cert != "" && key != "" {
			lab.InitWithClientCerts(h, u, t, ca, key, cert)
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
	default:
		return false
	}
}
