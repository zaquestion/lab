package cmd

import (
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/pkg/errors"
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
		lab label list remote "search term"
	`),
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, labelSearch, err := parseArgsRemoteAndProject(args)
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
			if label.Description != "" {
				description = " - " + label.Description
			}

			fmt.Printf("%s%s\n", label.Name, description)
		}
	},
}

func mapLabels(rn string, labelTerms []string) ([]string, error) {
	// Don't bother fetching project labels if nothing is being really requested
	if len(labelTerms) == 0 {
		return []string{}, nil
	}

	labels, err := lab.LabelList(rn)
	if err != nil {
		return nil, err
	}

	labelNames := make([]string, len(labels))
	for _, label := range labels {
		labelNames = append(labelNames, label.Name)
	}

	matches, err := matchTerms(labelTerms, labelNames)
	if err != nil {
		return nil, errors.Errorf("Label %s\n", err.Error())
	}

	return matches, nil
}

func init() {
	labelCmd.AddCommand(labelListCmd)
	carapace.Gen(labelCmd).PositionalCompletion(
		action.Remotes(),
	)
}
