package cmd

import (
	"github.com/MakeNowJust/heredoc/v2"
	lab "github.com/zaquestion/lab/internal/gitlab"
	"net/url"
	"path"
	"strconv"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
)

var snippetBrowseCmd = &cobra.Command{
	Use:   "browse [remote] <id>",
	Short: "View personal or project snippet in a browser",
	Example: heredoc.Doc(`
		lab snippet browse
		lab snippet browse upstream`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsRemoteAndID(args)
		if err != nil {
			log.Fatal(err)
		}

		host := lab.Host()
		hostURL, err := url.Parse(host)
		if err != nil {
			log.Fatal(err)
		}

		// See if we're in a git repo or if global is set to determine
		// if this should be a personal snippet
		if global || rn == "" {
			hostURL.Path = path.Join(hostURL.Path, "dashboard", "snippets")
		} else {
			hostURL.Path = path.Join(hostURL.Path, rn, "-", "snippets")
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
	carapace.Gen(snippetBrowseCmd).PositionalCompletion(
		action.Remotes(),
		action.Snippets(snippetList),
	)
}
