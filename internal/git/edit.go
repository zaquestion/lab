package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/shlex"
)

// Edit opens a file in the users editor and returns the title and body.
func Edit(filePrefix, message string) (string, string, error) {
	contents, err := EditFile(filePrefix, message)
	if err != nil {
		return "", "", err
	}

	return ParseTitleBody(strings.TrimSpace(string(contents)))
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
	editor, err := editor()
	if err != nil {
		return "", err
	}

	// Write generated/template message to file
	if _, err := os.Stat(filePath); os.IsNotExist(err) && message != "" {
		err = ioutil.WriteFile(filePath, []byte(message), 0644)
		if err != nil {
			fmt.Printf("ERROR(WriteFile): Saved file contents written to %s\n", filePath)
			return "", err
		}
	}

	cmd, err := editorCMD(editor, filePath)
	if err != nil {
		fmt.Printf("ERROR(editorCMD): failed to get editor command \"%s\"\n", editor)
		return "", err
	}

	err = cmd.Run()
	if err != nil {
		fmt.Printf("ERROR(cmd): Saved file contents written to %s\n", filePath)
		return "", err
	}

	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("ERROR(ReadFile): Saved file contents written to %s\n", filePath)
		return "", err
	}

	os.Remove(filePath)
	return removeComments(string(contents))
}

func editor() (string, error) {
	cmd := New("var", "GIT_EDITOR")
	cmd.Stdout = nil
	a, err := cmd.Output()
	if err != nil {
		return "", err
	}
	editor := strings.TrimSpace(string(a))
	if editor == "" {
		cmd = New("config", "--get", "core.editor")
		cmd.Stdout = nil
		b, err := cmd.Output()
		if err != nil {
			return "", err
		}
		editor = strings.TrimSpace(string(b))
	}
	return editor, nil
}

func editorCMD(editor, filePath string) (*exec.Cmd, error) {
	// make 'vi' the default editor to avoid empty editor configs
	if editor == "" {
		editor = "vi"
	}

	r := regexp.MustCompile("[nmg]?vi[m]?$")
	args := make([]string, 0, 3)
	if r.MatchString(editor) {
		args = append(args, "--cmd", "set ft=gitcommit tw=0 wrap lbr")
	}

	// Split editor command using shell rules for quoting and commenting
	parts, err := shlex.Split(editor)
	if err != nil {
		return nil, err
	}

	name := parts[0]
	if len(parts) > 0 {
		for _, arg := range parts[1:] {
			arg = strings.Replace(arg, "'", "\"", -1)
			args = append(args, arg)
		}
	}
	args = append(args, filePath)

	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
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

func ParseTitleBody(message string) (string, string, error) {
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
