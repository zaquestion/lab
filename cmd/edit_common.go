package cmd

import (
	"bufio"
	"fmt"
	"os"
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

// GetUpdateUsers returns an int slice of user IDs based on the
// current users and flags from the command line, and a bool
// indicating whether the users have changed
func getUpdateUsers(currentUsers []string, users []string, remove []string) ([]int, bool, error) {
	// add the new users to the current users, then remove the "remove" group
	users = difference(union(currentUsers, users), remove)
	usersChanged := !same(currentUsers, users)

	// turn the new user list into a list of user IDs
	var userIDs []int
	if usersChanged && len(users) == 0 {
		// if we're removing all users, we have to use []int{0}
		// see https://github.com/xanzy/go-gitlab/issues/427
		userIDs = []int{0}
	} else {
		userIDs = make([]int, len(users))
		for i, a := range users {
			if getUserID(a) == nil {
				return nil, false, fmt.Errorf("Error: %s is not a valid username\n", a)
			}
			userIDs[i] = *getUserID(a)
		}
	}

	return userIDs, usersChanged, nil
}

// editGetTitleDescription returns a title and description based on the
// current issue title and description and various flags from the command
// line
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

// editGetTitleDescFromFile returns the new title and description based on
// the content of a file. The first line is considered the title, the
// remaining is the description.
func editGetTitleDescFromFile(filename string) (string, string, error) {
	var title, body string

	file, err := os.Open(filename)
	if err != nil {
		return "", "", nil
	}
	defer file.Close()

	fileScan := bufio.NewScanner(file)
	fileScan.Split(bufio.ScanLines)

	// The first line in the file is the title.
	fileScan.Scan()
	title = fileScan.Text()

	for fileScan.Scan() {
		body = body + fileScan.Text() + "\n"
	}

	return title, body, nil
}
