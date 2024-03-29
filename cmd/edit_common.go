package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/zaquestion/lab/internal/git"
)

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
				return nil, false, fmt.Errorf("%s is not a valid username", a)
			}
			userIDs[i] = *getUserID(a)
		}
	}

	return userIDs, usersChanged, nil
}

// editDescription returns a title and description based on the
// current issue title and description and various flags from the command
// line
func editDescription(title string, body string, msgs []string, filename string) (string, string, error) {
	if len(msgs) > 0 {
		title = msgs[0]

		if len(msgs) > 1 {
			body = strings.Join(msgs[1:], "\n\n")
		}

		return title, body, nil
	}

	if filename != "" {
		var lines []string

		content, err := ioutil.ReadFile(filename)
		if err != nil {
			return "", "", err
		}
		lines = strings.Split(string(content), "\n")

		title = lines[0]
		body = strings.Join(lines[1:], "\n")

		return title, body, nil
	}

	text, err := editText(title, body)
	if err != nil {
		return "", "", err
	}

	title, body, err = git.Edit("EDIT", text)
	if err != nil {
		_, f, l, _ := runtime.Caller(0)
		log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
	}

	return title, body, nil
}

// editText places the text title and body in a specific template following Git
// standards.
func editText(title string, body string) (string, error) {
	tmpl := heredoc.Doc(`
		{{.InitMsg}}

		{{.CommentChar}} Edit the title and/or description. The first block of text
		{{.CommentChar}} is the title and the rest is the description.`)

	msg := &struct {
		InitMsg     string
		CommentChar string
	}{
		InitMsg:     title + "\n\n" + body,
		CommentChar: git.CommentChar(),
	}

	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	err = t.Execute(&b, msg)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}
