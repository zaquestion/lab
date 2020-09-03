package main

import (
	"log"
	"os"

	"github.com/rsteube/carapace"
	"github.com/spf13/viper"
	"github.com/zaquestion/lab/cmd"
	"github.com/zaquestion/lab/internal/config"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// version gets set on releases during build by goreleaser.
var version = "master"

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

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	cmd.Version = version
	if !skipInit() {
		ca := loadTLSCerts()
		h, u, t, skipVerify := config.LoadConfig()

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
