package cmd

import (
	"log"
	"net/url"
	"path"
	"strconv"

	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	git "github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
	zsh "github.com/rsteube/cobra-zsh-gen"
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

		host := lab.Host()
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
			if len(mrs) > 0 {
				num = int64(mrs[0].IID)
				hostURL.Path = path.Join(hostURL.Path, strconv.FormatInt(num, 10))
			}
		}

		err = browse(hostURL.String())
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	zsh.Wrap(mrBrowseCmd).MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	zsh.Wrap(mrBrowseCmd).MarkZshCompPositionalArgumentCustom(2, "__lab_completion_merge_request $words[2]")
	mrCmd.AddCommand(mrBrowseCmd)
}
