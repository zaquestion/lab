package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Edit opens a file in the users editor and returns the title and body.
func Edit(filePrefix, message string) (string, string, error) {
	contents, err := EditFile(filePrefix, message)
	if err != nil {
		return "", "", err
	}

	return parseTitleBody(strings.TrimSpace(string(contents)))
}

// EditFile opens a file in the users editor and returns the contents. It
// stores a temporary file in your .git directory or /tmp if accessed outside of
// a git repo.
func EditFile(filePrefix, message string) (string, error) {
	var (
		dir string
		err error
	)
	if InsideGitRepo() {
		dir, err = Dir()
		if err != nil {
			return "", err
		}
	} else {
		dir = "/tmp"
	}
	filePath := filepath.Join(dir, fmt.Sprintf("%s_EDITMSG", filePrefix))
	editorPath, err := editorPath()
	if err != nil {
		return "", err
	}
	defer os.Remove(filePath)

	// Write generated/template message to file
	if _, err := os.Stat(filePath); os.IsNotExist(err) && message != "" {
		err = ioutil.WriteFile(filePath, []byte(message), 0644)
		if err != nil {
			return "", err
		}
	}

	cmd := editorCMD(editorPath, filePath)
	err = cmd.Run()
	if err != nil {
		return "", err
	}

	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return removeComments(string(contents))
}

func editorPath() (string, error) {
	cmd := New("var", "GIT_EDITOR")
	cmd.Stdout = nil
	e, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(e)), nil
}

func editorCMD(editorPath, filePath string) *exec.Cmd {
	parts := strings.Split(editorPath, " ")
	r := regexp.MustCompile("[nmg]?vi[m]?$")
	args := make([]string, 0, 3)
	if r.MatchString(editorPath) {
		args = append(args, "--cmd", "set ft=gitcommit tw=0 wrap lbr")
	}
	argparts := strings.Join(parts[1:], " ")
	argparts = strings.Replace(argparts, "'", "\"", -1)
	args = append(args, argparts, filePath)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func removeComments(message string) (string, error) {
	// Grab all the lines that don't start with the comment char
	cc := CommentChar()
	r := regexp.MustCompile(`(?m:^)[^` + cc + `].*(?m:$)`)
	cr := regexp.MustCompile(`(?m:^)\s*` + cc)
	parts := r.FindAllString(message, -1)
	noComments := make([]string, 0)
	for _, p := range parts {
		if !cr.MatchString(p) {
			noComments = append(noComments, p)
		}
	}
	return strings.TrimSpace(strings.Join(noComments, "\n")), nil
}

func parseTitleBody(message string) (string, string, error) {
	msg, err := removeComments(message)
	if err != nil {
		return "", "", err
	}

	if msg == "" {
		return "", "", nil
	}

	r := regexp.MustCompile(`\n\s*\n`)
	msg = strings.Replace(msg, "\\#", "#", -1)
	parts := r.Split(msg, 2)

	if strings.Contains(parts[0], "\n") {
		return "\n", parts[0], nil
	}
	if len(parts) < 2 {
		return parts[0], "", nil
	}
	return parts[0], parts[1], nil
}
