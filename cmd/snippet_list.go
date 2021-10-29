package cmd

import (
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"
	"strconv"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var snippetListConfig struct {
	Number string
	All    bool
}

// snippetListCmd represents the snippetList command
var snippetListCmd = &cobra.Command{
	Use:     "list [remote]",
	Aliases: []string{"ls"},
	Short:   "List personal or project snippets",
	Example: heredoc.Doc(`
		lab snippet list
		lab snippet list -a
		lab snippet list -n 10
		lab snippet list -m "Snippet example" -M "Description message"
		lab snippet list upstream --private
		lab snippet list origin --public`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		snips, err := snippetList(args)
		if err != nil {
			log.Fatal(err)
		}
		pager := newPager(cmd.Flags())
		defer pager.Close()
		for _, snip := range snips {
			fmt.Printf("#%d %s\n", snip.ID, snip.Title)
		}
	},
}

func snippetList(args []string) ([]*gitlab.Snippet, error) {
	rn, _, err := parseArgsRemoteAndID(args)
	if err != nil {
		return nil, err
	}

	num, err := strconv.Atoi(snippetListConfig.Number)
	if snippetListConfig.All || (err != nil) {
		num = -1
	}

	listOpts := gitlab.ListOptions{
		PerPage: num,
	}

	// See if we're in a git repo or if global is set to determine
	// if this should be a personal snippet
	if global || rn == "" {
		opts := gitlab.ListSnippetsOptions(listOpts)
		return lab.SnippetList(opts, num)
	}

	opts := gitlab.ListProjectSnippetsOptions(listOpts)
	return lab.ProjectSnippetList(rn, opts, num)
}

func init() {
	snippetListCmd.Flags().StringVarP(&snippetListConfig.Number, "number", "n", "10", "Number of snippets to return")
	snippetListCmd.Flags().BoolVarP(&snippetListConfig.All, "all", "a", false, "list all snippets")

	snippetCmd.AddCommand(snippetListCmd)
	carapace.Gen(snippetListCmd).PositionalCompletion(
		action.Remotes(),
	)
}
