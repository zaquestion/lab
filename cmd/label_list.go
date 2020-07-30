package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var labelListCmd = &cobra.Command{
	Use:     "list [remote] [search]",
	Aliases: []string{"ls", "search"},
	Short:   "List labels",
	Long:    ``,
	Example: `lab label list                       # list all labels
lab label list "search term"         # search labels for "search term"
lab label search "search term"       # same as above
lab label list remote "search term"  # search "remote" for labels with "search term"`,
	Run: func(cmd *cobra.Command, args []string) {
		rn, labelSearch, err := parseArgsRemoteString(args)
		if err != nil {
			log.Fatal(err)
		}

		labelSearch = strings.ToLower(labelSearch)

		labels, err := lab.LabelList(rn)
		if err != nil {
			log.Fatal(err)
		}

		for _, label := range labels {
			// GitLab API has no search for labels, so we do it ourselves
			if labelSearch != "" &&
				!(strings.Contains(strings.ToLower(label.Name), labelSearch) || strings.Contains(strings.ToLower(label.Description), labelSearch)) {
				continue
			}

			description := ""
			if label.Description != "" {
				description = " - " + label.Description
			}

			fmt.Printf("%s%s\n", label.Name, description)
		}
	},
}

func init() {
	labelCmd.AddCommand(labelListCmd)
	carapace.Gen(labelCmd).PositionalCompletion(
		action.Remotes(),
	)
}
