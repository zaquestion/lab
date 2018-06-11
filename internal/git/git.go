package git

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	retry "github.com/avast/retry-go"
	"github.com/pkg/errors"
	gitconfig "github.com/tcnksm/go-gitconfig"
)

// IsHub is true when using "hub" as the git binary
var IsHub bool

func init() {
	_, err := exec.LookPath("hub")
	if err == nil {
		IsHub = true
	}
}

// New looks up the hub or git binary and returns a cmd which outputs to stdout
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

// GitDir returns the full path to the .git directory
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

// WorkingDir returns the full pall to the root of the current git repository
func WorkingDir() (string, error) {
	cmd := New("rev-parse", "--show-toplevel")
	cmd.Stdout = nil
	d, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(d)), nil
}

// CommentChar returns active comment char and defaults to '#'
func CommentChar() string {
	char, err := gitconfig.Entire("core.commentchar")
	if err == nil {
		return char
	}
	return "#"
}

// LastCommitMessage returns the last commits message as one line
func LastCommitMessage() (string, error) {
	cmd := New("show", "-s", "--format=%s%n%+b", "HEAD")
	cmd.Stdout = nil
	msg, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(msg)), nil
}

// Log produces a formatted gitlog between 2 git shas
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

// CurrentBranch returns the currently checked out branch and strips away all
// but the branchname itself.
func CurrentBranch() (string, error) {
	cmd := New("branch")
	cmd.Stdout = nil
	gBranches, err := cmd.Output()
	if err != nil {
		return "", err
	}
	branches := strings.Split(string(gBranches), "\n")
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

// PathWithNameSpace returns the owner/repository for the current repo
// Such as zaquestion/lab
// Respects GitLab subgroups (https://docs.gitlab.com/ce/user/group/subgroups/)
func PathWithNameSpace(remote string) (string, error) {
	remoteURL, err := gitconfig.Local("remote." + remote + ".url")
	if err != nil {
		return "", err
	}

	parts := strings.Split(remoteURL, "//")

	if len(parts) == 1 {
		// scp-like short syntax (e.g. git@gitlab.com...)
		part := parts[0]
		parts = strings.Split(part, ":")
	} else if len(parts) == 2 {
		// every other protocol syntax (e.g. ssh://, http://, git://)
		part := parts[1]
		parts = strings.SplitN(part, "/", 2)
	} else {
		return "", errors.Errorf("cannot parse remote: %s url: %s", remote, remoteURL)
	}

	if len(parts) != 2 {
		return "", errors.Errorf("cannot parse remote: %s url: %s", remote, remoteURL)
	}
	path := parts[1]
	path = strings.TrimSuffix(path, ".git")
	return path, nil
}

// RepoName returns the name of the repository, such as "lab"
func RepoName() (string, error) {
	o, err := PathWithNameSpace("origin")
	if err != nil {
		return "", err
	}
	parts := strings.Split(o, "/")
	return parts[len(parts)-1:][0], nil
}

// RemoteAdd both adds a remote and fetches it
func RemoteAdd(name, url, dir string) error {
	cmd := New("remote", "add", name, url)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}
	fmt.Println("Updating", name)

	err := retry.Do(func() error {
		cmd = New("fetch", name)
		cmd.Dir = dir
		return cmd.Run()
	}, retry.Attempts(3), retry.Delay(time.Second), retry.Units(time.Nanosecond))
	if err != nil {
		return err
	}
	fmt.Println("new remote:", name)
	return nil
}

// IsRemote returns true when passed a valid remote in the git repo
func IsRemote(remote string) (bool, error) {
	cmd := New("remote")
	cmd.Stdout = nil
	remotes, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return bytes.Contains(remotes, []byte(remote+"\n")), nil
}

// InsideGitRepo returns true when the current working directory is inside the
// working tree of a git repo
func InsideGitRepo() bool {
	cmd := New("rev-parse", "--is-inside-work-tree")
	cmd.Stdout = nil
	cmd.Stderr = nil
	out, _ := cmd.CombinedOutput()
	return bytes.Contains(out, []byte("true\n"))
}
