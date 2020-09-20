// This file contains common functions that are shared in the lab package
package cmd

import (
	"log"
	"strconv"
	"strings"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/zaquestion/lab/internal/config"
)

var (
	CommandPrefix string
)

// flagConfig compares command line flags and the flags set in the config
// files.  The command line value will always override any value set in the
// config files.
func flagConfig(fs *flag.FlagSet) {
	fs.VisitAll(func(f *flag.Flag) {
		var (
			configValue  interface{}
			configString string
		)

		switch f.Value.Type() {
		case "bool":
			configValue = getMainConfig().GetBool(CommandPrefix + f.Name)
			configString = strconv.FormatBool(configValue.(bool))
		case "string":
			configValue = getMainConfig().GetString(CommandPrefix + f.Name)
			configString = configValue.(string)
		case "stringSlice":
			configValue = getMainConfig().GetStringSlice(CommandPrefix + f.Name)
			configString = strings.Join(configValue.([]string), " ")

		case "int":
			configValue = getMainConfig().GetInt64(CommandPrefix + f.Name)
			configString = strconv.FormatInt(configValue.(int64), 10)
		case "stringArray":
			// viper does not have support for stringArray
			configString = ""
		default:
			log.Fatal("ERROR: found unidentified flag: ", f.Value.Type(), f)
		}

		// if set, always use the command line option (flag) value
		if f.Value.String() != f.DefValue {
			return
		}
		// o/w use the value in the configfile
		if configString != "" && configString != f.DefValue {
			f.Value.Set(configString)
		}
	})
}

// getMainConfig returns the merged config of ~/.config/lab/lab.toml and
// .git/lab/lab.toml
func getMainConfig() *viper.Viper {
	return config.MainConfig
}

// textToMarkdown converts text with markdown friendly line breaks
// See https://gist.github.com/shaunlebron/746476e6e7a4d698b373 for more info.
func textToMarkdown(text string) string {
	text = strings.Replace(text, "\n", "  \n", -1)
	return text
}
