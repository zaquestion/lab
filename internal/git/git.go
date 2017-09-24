package git

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/tcnksm/go-gitconfig"
)

var IsHub bool

func init() {
	_, err := exec.LookPath("hub")
	if err == nil {
		IsHub = true
	}
}

func New(args ...string) *exec.Cmd {
	gitPath, err := exec.LookPath("hub")
	if err != nil {
		gitPath, err = exec.LookPath("git")
		if err != nil {
			log.Fatal(err)
		}
	}

	cmd := exec.Command(gitPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func GitDir() (string, error) {
	cmd := New("rev-parse", "-q", "--git-dir")
	cmd.Stdout = nil
	d, err := cmd.Output()
	if err != nil {
		return "", err
	}
	dir := string(d)
	dir = strings.TrimSpace(dir)
	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			return "", err
		}
	}

	return filepath.Clean(dir), nil
}

func WorkingDir() (string, error) {
	cmd := New("rev-parse", "--show-toplevel")
	cmd.Stdout = nil
	d, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(d)), nil
}

func CommentChar() string {
	char, err := gitconfig.Local("core.commentchar")
	if err == nil {
		return char
	}

	char, err = gitconfig.Global("core.commentchar")
	if err == nil {
		return char
	}

	return "#"
}

func LastCommitMessage() (string, error) {
	cmd := New("show", "-s", "--format=%s%n%+b", "HEAD")
	cmd.Stdout = nil
	msg, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(msg)), nil
}

func Log(sha1, sha2 string) (string, error) {
	cmd := New("-c", "log.showSignature=false",
		"log",
		"--no-color",
		"--format=%h (%aN, %ar)%n%w(78,3,3)%s%n",
		"--cherry",
		fmt.Sprintf("%s...%s", sha1, sha2))
	cmd.Stdout = nil
	outputs, err := cmd.Output()
	if err != nil {
		return "", errors.Errorf("Can't load git log %s..%s", sha1, sha2)
	}

	return string(outputs), nil
}

func CurrentBranch() (string, error) {
	cmd := New("branch")
	cmd.Stdout = nil
	gBranches, err := cmd.Output()
	if err != nil {
		return "", err
	}
	branches := strings.Split(string(gBranches), "\n")
	if os.Getenv("DEBUG") != "" {
		spew.Dump(branches)
	}
	var branch string
	for _, b := range branches {
		if strings.HasPrefix(b, "* ") {
			branch = b
			break
		}
	}
	if branch == "" {
		return "", errors.New("current branch could not be determined")
	}
	branch = strings.TrimPrefix(branch, "* ")
	branch = strings.TrimSpace(branch)
	return branch, nil
}

func PathWithNameSpace(remote string) (string, error) {
	remoteURL, err := gitconfig.Local("remote." + remote + ".url")
	if err != nil {
		return "", err
	}
	parts := strings.Split(remoteURL, ":")
	if len(parts) == 0 {
		return "", errors.New("remote." + remote + ".url missing repository")
	}
	return strings.TrimSuffix(parts[len(parts)-1:][0], ".git"), nil
}

func RepoName() (string, error) {
	o, err := PathWithNameSpace("origin")
	if err != nil {
		return "", err
	}
	parts := strings.Split(o, "/")
	return parts[len(parts)-1:][0], nil
}

func RemoteAdd(name, url string) error {
	err := New("remote", "add", name, url).Run()
	if err != nil {
		return err
	}
	fmt.Println("Updating", name)
	err = New("fetch", name).Run()
	if err != nil {
		return err
	}
	fmt.Println("new remote:", name)
	return nil
}
