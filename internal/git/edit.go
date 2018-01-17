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

// Edit opens a file in the users editor and returns the title and body. It
// store a temporary file in your .git directory or /tmp if accessed outside of
// a git repo.
func Edit(filePrefix, message string) (string, string, error) {
	var (
		dir string
		err error
	)
	if InsideGitRepo() {
		dir, err = GitDir()
		if err != nil {
			return "", "", err
		}
	} else {
		dir = "/tmp"
	}
	filePath := filepath.Join(dir, fmt.Sprintf("%s_EDITMSG", filePrefix))
	editorPath, err := editorPath()
	if err != nil {
		return "", "", err
	}
	defer os.Remove(filePath)

	// Write generated/template message to file
	if _, err := os.Stat(filePath); os.IsNotExist(err) && message != "" {
		err = ioutil.WriteFile(filePath, []byte(message), 0644)
		if err != nil {
			return "", "", err
		}
	}

	cmd := editorCMD(editorPath, filePath)
	err = cmd.Run()
	if err != nil {
		return "", "", err
	}

	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", "", err
	}

	return parseTitleBody(strings.TrimSpace(string(contents)))
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
	args = append(args, parts[1:]...)
	args = append(args, filePath)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func parseTitleBody(message string) (string, string, error) {
	// Grab all the lines that don't start with the comment char
	cc := CommentChar()
	r := regexp.MustCompile(`(?m:^)[^` + cc + `].*(?m:$)`)
	cr := regexp.MustCompile(`(?m:^)\s*#`)
	parts := r.FindAllString(message, -1)
	noComments := make([]string, 0)
	for _, p := range parts {
		if !cr.MatchString(p) {
			noComments = append(noComments, p)
		}
	}
	msg := strings.Join(noComments, "\n")
	if strings.TrimSpace(msg) == "" {
		return "", "", nil
	}

	r = regexp.MustCompile(`\n\s*\n`)
	parts = r.Split(msg, 2)
	title := strings.Replace(parts[0], "\n", " ", -1)
	if len(parts) < 2 {
		return title, "", nil
	}
	return title, parts[1], nil
}
