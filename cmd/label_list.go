package cmd

import (
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var labelListCmd = &cobra.Command{
	Use:     "list [remote] [search]",
	Aliases: []string{"ls", "search"},
	Short:   "List labels",
	Example: heredoc.Doc(`
		lab label list
		lab label list "search term"
		lab label list remote "search term"`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, labelSearch, err := parseArgsRemoteAndProject(args)
		if err != nil {
			log.Fatal(err)
		}

		nameOnly, err := cmd.Flags().GetBool("name-only")
		if err != nil {
			log.Fatal(err)
		}

		color, err := cmd.Flags().GetBool("color")
		if err != nil {
			log.Fatal(err)
		}

		labelSearch = strings.ToLower(labelSearch)

		labels, err := lab.LabelList(rn)
		if err != nil {
			log.Fatal(err)
		}

		pager := newPager(cmd.Flags())
		defer pager.Close()

		for _, label := range labels {
			// GitLab API has no search for labels, so we do it ourselves
			if labelSearch != "" &&
				!(strings.Contains(strings.ToLower(label.Name), labelSearch) || strings.Contains(strings.ToLower(label.Description), labelSearch)) {
				continue
			}

			description := ""
			if !nameOnly && label.Description != "" {
				description = " - " + label.Description
			}

			// Default format without color
			format := "%s%s\n"
			if color {
				// Convert hex color to rgb object
				c := HexToRGB(label.Color)
				format = fmt.Sprintf("\033[48;2;%d;%d;%dm%%s\033[0m%%s\n", c.R, c.G, c.B)
			}

			fmt.Printf(format, label.Name, description)
		}
	},
}

func init() {
	labelListCmd.Flags().Bool("name-only", false, "only list label names, not descriptions")
	labelListCmd.Flags().Bool("color", false, "print colored labels")
	labelCmd.AddCommand(labelListCmd)
	carapace.Gen(labelCmd).PositionalCompletion(
		action.Remotes(),
	)
}
