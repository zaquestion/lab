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

var mrBrowseCmd = &cobra.Command{
	Use:     "browse [remote]",
	Aliases: []string{"b"},
	Short:   "browse merge requests",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		remote, num, err := parseArgsRemote(args)
		if err != nil {
			log.Fatal(err)
		}
		if remote == "" {
			remote = forkedFromRemote
		}
		rn, err := git.PathWithNameSpace(remote)
		if err != nil {
			log.Fatal(err)
		}

		c := viper.AllSettings()["core"]
		config := c.([]map[string]interface{})[0]
		host := config["host"].(string)

		hostURL, err := url.Parse(host)
		if err != nil {
			log.Fatal(err)
		}
		hostURL.Path = path.Join(hostURL.Path, rn, "merge_requests")
		if num > 0 {
			hostURL.Path = path.Join(hostURL.Path, strconv.FormatInt(num, 10))
		}

		err = browse(hostURL.String())
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	mrCmd.AddCommand(mrBrowseCmd)
}
