package cmd

import (
	"log"
	"net/url"
	"path"
	"strconv"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrBrowseCmd = &cobra.Command{
	Use:              "browse [remote] <id>",
	Aliases:          []string{"b"},
	Short:            "View merge request in a browser",
	Long:             ``,
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, num, err := parseArgsWithGitBranchMR(args)
		if err != nil {
			log.Fatal(err)
		}

		host := lab.Host()
		hostURL, err := url.Parse(host)
		if err != nil {
			log.Fatal(err)
		}
		hostURL.Path = path.Join(hostURL.Path, rn, "merge_requests")
		hostURL.Path = path.Join(hostURL.Path, strconv.FormatInt(num, 10))

		err = browse(hostURL.String())
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	mrCmd.AddCommand(mrBrowseCmd)
	carapace.Gen(mrBrowseCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
