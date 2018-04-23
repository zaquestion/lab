package main

import (
	"log"
	"os"
	"path"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"github.com/zaquestion/lab/cmd"
	"github.com/zaquestion/lab/internal/config"
	lab "github.com/zaquestion/lab/internal/gitlab"
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
	if _, ok := viper.ReadInConfig().(viper.ConfigFileNotFoundError); ok {
		if err := config.New(path.Join(confpath, "lab.hcl"), os.Stdin); err != nil {
			log.Fatal(err)
		}

		if err := viper.ReadInConfig(); err != nil {
			log.Fatal(err)
		}
	}

	c := viper.AllSettings()["core"]
	var cfg map[string]interface{}
	switch v := c.(type) {
	// Most run this is the type
	case []map[string]interface{}:
		cfg = v[0]
	// On the first run when the cfg is created it comes in as this type
	// for whatever reason
	case map[string]interface{}:
		cfg = v
	}

	for _, v := range []string{"host", "user", "token"} {
		if cv, ok := cfg[v]; !ok {
			log.Println(cv)
			log.Fatalf("missing config value core.%s in %s", v, viper.ConfigFileUsed())
		}
	}

	lab.Init(
		cfg["host"].(string),
		cfg["user"].(string),
		cfg["token"].(string))

	cmd.Execute()
}
