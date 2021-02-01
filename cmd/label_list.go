package cmd

import (
	"fmt"
	"log"
	"strings"

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
	Long:    ``,
	Example: `lab label list                       # list all labels
lab label list "search term"         # search labels for "search term"
lab label search "search term"       # same as above
lab label list remote "search term"  # search "remote" for labels with "search term"`,
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

		pager := NewPager(cmd.Flags())
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

func MapLabels(rn string, labelTerms []string) ([]string, error) {
	labels, err := lab.LabelList(rn)
	if err != nil {
		return nil, err
	}

	matches := make([]string, len(labelTerms))
	for i, term := range labelTerms {
		lowerTerm := strings.ToLower(term)
		for _, label := range labels {
			lowerLabel := strings.ToLower(label.Name)

			// no match, or we already have an exact match
			if !strings.Contains(lowerLabel, lowerTerm) || matches[i] == lowerLabel {
				continue
			}

			// only allow ambiguity for exact matches
			if matches[i] != "" && lowerTerm != lowerLabel {
				return nil, errors.Errorf("Label '%s' is ambiguous", term)
			}

			matches[i] = label.Name
		}

		if matches[i] == "" {
			return nil, errors.Errorf("Label '%s' not found", term)
		}
	}

	return matches, nil
}

func init() {
	labelCmd.AddCommand(labelListCmd)
	carapace.Gen(labelCmd).PositionalCompletion(
		action.Remotes(),
	)
}
