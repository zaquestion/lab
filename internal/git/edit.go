package git

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func Edit(filePrefix, message string) (string, string, error) {
	gitDir, err := Dir()
	if err != nil {
		return "", "", err
	}
	filePath := filepath.Join(gitDir, fmt.Sprintf("%s_EDITMSG", filePrefix))
	if os.Getenv("DEBUG") != "" {
		log.Println("msgFile:", filePath)
	}

	editorPath, err := EditorPath()
	if err != nil {
		return "", "", err
	}
	defer os.Remove(filePath)

	// Write generated/tempate message to file
	if _, err := os.Stat(filePath); os.IsNotExist(err) && message != "" {
		err = ioutil.WriteFile(filePath, []byte(message), 0644)
		if err != nil {
			return "", "", err
		}
	}

	err = openTextEditor(editorPath, filePath)
	if err != nil {
		return "", "", err
	}

	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", "", err
	}

	return parseTitleBody(strings.TrimSpace(string(contents)))
}

func openTextEditor(editorPath, filePath string) error {
	r := regexp.MustCompile("[nmg]?vi[m]?$")
	args := make([]string, 0, 3)
	if r.MatchString(editorPath) {
		args = append(args, "--cmd", "set ft=gitcommit tw=0 wrap lbr")
	}
	args = append(args, filePath)
	cmd := exec.Command(editorPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
