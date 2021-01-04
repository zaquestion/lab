package cmd

import (
	"strings"

	"github.com/zaquestion/lab/internal/git"
)

// editGetLabels returns a string slice of labels based on the current
// labels and flags from the command line, and a bool indicating whether
// the labels have changed
func editGetLabels(idLabels []string, labels []string, unlabels []string) ([]string, bool, error) {
	// add the new labels to the current labels, then remove the "unlabels"
	labels = difference(union(idLabels, labels), unlabels)

	return labels, !same(idLabels, labels), nil
}

// GetUpdateAssignees returns an int slice of assignee IDs based on the
// current assignees and flags from the command line, and a bool
// indicating whether the assignees have changed
func getUpdateAssignees(currentAssignees []string, assignees []string, unassignees []string) ([]int, bool, error) {
	// add the new assignees to the current assignees, then remove the "unassignees"
	assignees = difference(union(currentAssignees, assignees), unassignees)
	assigneesChanged := !same(currentAssignees, assignees)

	// turn the new assignee list into a list of assignee IDs
	var assigneeIDs []int
	if assigneesChanged && len(assignees) == 0 {
		// if we're removing all assignees, we have to use []int{0}
		// see https://github.com/xanzy/go-gitlab/issues/427
		assigneeIDs = []int{0}
	} else {
		assigneeIDs = make([]int, len(assignees))
		for i, a := range assignees {
			assigneeIDs[i] = *getAssigneeID(a)
		}
	}

	return assigneeIDs, assigneesChanged, nil
}

// editGetTitleDescription returns a title and description based on the current
// issue title and description and various flags from the command line
func editGetTitleDescription(title string, body string, msgs []string, nFlag int) (string, string, error) {
	if len(msgs) > 0 {
		title = msgs[0]

		if len(msgs) > 1 {
			body = strings.Join(msgs[1:], "\n\n")
		}

		// we have everything we need
		return title, body, nil
	}

	// if other flags were given (eg label), then skip the editor and return
	// what we already have
	if nFlag != 0 {
		return title, body, nil
	}

	text, err := editText(title, body)
	if err != nil {
		return "", "", err
	}
	return git.Edit("EDIT", text)
}
