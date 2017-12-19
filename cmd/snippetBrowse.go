package cmd

import (
	"log"
	"net/url"
	"path"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zaquestion/lab/internal/git"
)

var snippetBrowseCmd = &cobra.Command{
	Use:   "browse [remote]",
	Short: "browse personal or project snippets",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		remote, id, err := parseArgsRemote(args)
		if err != nil {
			log.Fatal(err)
		}
		if remote == "" {
			remote = forkedFromRemote
		}

		c := viper.AllSettings()["core"]
		config := c.([]map[string]interface{})[0]
		host := config["host"].(string)
		hostURL, err := url.Parse(host)
		if err != nil {
			log.Fatal(err)
		}

		// See if we're in a git repo or if global is set to determine
		// if this should be a personal snippet
		rn, _ := git.PathWithNameSpace(remote)
		if global || rn == "" {
			hostURL.Path = path.Join(hostURL.Path, "dashboard", "snippets")
		} else {
			hostURL.Path = path.Join(hostURL.Path, rn, "snippets")
		}

		if id > 0 {
			hostURL.Path = path.Join(hostURL.Path, strconv.FormatInt(id, 10))
		}

		err = browse(hostURL.String())
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	snippetCmd.AddCommand(snippetBrowseCmd)
}
