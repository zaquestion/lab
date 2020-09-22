// This file contains common functions that are shared in the lab package
package cmd

import (
	"path"
	"runtime"
	"strings"

	"github.com/spf13/viper"
	"github.com/zaquestion/lab/internal/config"
)

var (
	CommandPrefix string
)

// getMainConfig returns the merged config of ~/.config/lab/lab.toml and
// .git/lab/lab.toml
func getMainConfig() *viper.Viper {
	return config.MainConfig
}

// setCommandPrefix sets command name that is used in the config
// files to set per-command options.  For example, the "lab issue show"
// command has a prefix of "issue_show.", and "lab mr list" as a
// prefix of "mr_list."
func setCommandPrefix() {
	_, file, _, _ := runtime.Caller(1)
	_, filename := path.Split(file)
	s := strings.Split(filename, ".")
	CommandPrefix = s[0] + "."
}

// textToMarkdown converts text with markdown friendly line breaks
// See https://gist.github.com/shaunlebron/746476e6e7a4d698b373 for more info.
func textToMarkdown(text string) string {
	text = strings.Replace(text, "\n", "  \n", -1)
	return text
}
