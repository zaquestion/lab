package cmd

import (
	"fmt"
	"log"
	"net/url"
	"path"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xanzy/go-gitlab"
	git "github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrBrowseCmd = &cobra.Command{
	Use:     "browse [remote] <id>",
	Aliases: []string{"b"},
	Short:   "View merge request in a browser",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		rn, num, err := parseArgs(args)
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
		} else {
			currentBranch, err := git.CurrentBranch()
			if err != nil {
				log.Fatal(err)
			}
			mrs, err := lab.MRList(rn, gitlab.ListProjectMergeRequestsOptions{
				ListOptions: gitlab.ListOptions{
					PerPage: 10,
				},
				Labels:       mrLabels,
				State:        &mrState,
				OrderBy:      gitlab.String("updated_at"),
				SourceBranch: gitlab.String(currentBranch),
			}, -1)
			if err != nil {
				log.Fatal(err)
			}
			for _, mr := range mrs {
				fmt.Printf("#%d %s\n", mr.IID, mr.Title, mr.SourceBranch)
			}
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
