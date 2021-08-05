package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	retry "github.com/avast/retry-go"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pkg/errors"
	gitconfig "github.com/tcnksm/go-gitconfig"
	"github.com/zaquestion/lab/internal/logger"
)

// Get internal lab logger instance
var log = logger.GetInstance()

// New looks up the git binary and returns a cmd which outputs to stdout
func New(args ...string) *exec.Cmd {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command(gitPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// Dir returns the full path to the .git directory
func Dir() (string, error) {
	cmd := New("rev-parse", "-q", "--git-dir")
	cmd.Stdout = nil
	cmd.Stderr = nil
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

// WorkingDir returns the full path to the root of the current git repository
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

// PagerCommand returns the commandline and environment for the pager
func PagerCommand() (string, []string) {
	// Set up environment for common pagers, see the documentation
	// for "core.pager" in git-config(1)
	env := os.Environ()
	if _, ok := os.LookupEnv("LESS"); !ok {
		env = append(env, "LESS=FRX")
	}
	if _, ok := os.LookupEnv("LESSSECURE"); !ok {
		env = append(env, "LESSSECURE=1")
	}
	if _, ok := os.LookupEnv("LV"); !ok {
		env = append(env, "LV=-c")
	}

	// Find an appropriate pager command, following git's preference
	cmd, ok := os.LookupEnv("GIT_PAGER")
	if ok {
		return cmd, env
	}
	cmd, err := gitconfig.Entire("core.pager")
	if err == nil {
		return cmd, env
	}
	cmd, ok = os.LookupEnv("PAGER")
	if ok {
		return cmd, env
	}
	return "less", env
}

// LastCommitMessage returns the last commits message as one line
func LastCommitMessage(sha string) (string, error) {
	cmd := New("show", "-s", "--format=%s%n%+b", sha)
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
		"--format=%h (%aN)%n%w(78,3,3)%s%n",
		"--cherry",
		fmt.Sprintf("%s..%s", sha1, sha2))
	cmd.Stdout = nil
	outputs, err := cmd.Output()
	if err != nil {
		return "", errors.Errorf("Can't load git log %s..%s", sha1, sha2)
	}

	diffCmd := New("diff", "--stat", fmt.Sprintf("%s...%s", sha1, sha2))
	diffCmd.Stdout = nil
	diffOutput, err := diffCmd.Output()
	if err != nil {
		return "", errors.Errorf("Can't load diffstat")
	}

	return string(outputs) + string(diffOutput), nil
}

// CurrentBranch returns the currently checked out branch
func CurrentBranch() (string, error) {
	cmd := New("rev-parse", "--abbrev-ref", "HEAD")
	cmd.Stdout = nil
	branch, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(branch)), nil
}

// UpstreamBranch returns the upstream of the specified branch
func UpstreamBranch(branch string) (string, error) {
	upstreamBranch, err := gitconfig.Local("branch." + branch + ".merge")
	if err != nil {
		return "", errors.Errorf("No upstream for branch '%s'", branch)
	}
	return strings.TrimPrefix(upstreamBranch, "refs/heads/"), nil
}

// PathWithNamespace returns the owner/repository for the current repo
// Such as zaquestion/lab
// Respects GitLab subgroups (https://docs.gitlab.com/ce/user/group/subgroups/)
func PathWithNamespace(remote string) (string, error) {
	remoteURL, err := gitconfig.Local("remote." + remote + ".pushurl")
	if err != nil || remoteURL == "" {
		remoteURL, err = gitconfig.Local("remote." + remote + ".url")
		if err != nil {
			return "", err
		}
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
	o, err := PathWithNamespace("origin")
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
	}, retry.Attempts(3), retry.Delay(time.Second))
	if err != nil {
		return err
	}
	fmt.Println("new remote:", name)
	return nil
}

// Remotes get the list of remotes available in the current repo dir
func Remotes() ([]string, error) {
	repo, err := gogit.PlainOpen(".")
	if err != nil {
		return []string{}, err
	}
	remotes, err := repo.Remotes()
	if err != nil {
		return []string{}, err
	}

	names := make([]string, len(remotes))
	for i, r := range remotes {
		names[i] = r.Config().Name
	}
	return names, nil
}

// RemoteBranches get the list of branches the specified remote has
func RemoteBranches(remote string) ([]string, error) {
	repo, err := gogit.PlainOpen(".")
	if err != nil {
		return []string{}, err
	}

	branches, err := repo.References() // TODO verify is a branch Branches didn't seem to work
	if err != nil {
		return []string{}, err
	}
	reg := regexp.MustCompile(`^refs/remotes/[^/]+/`)

	names := []string{}
	branches.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsRemote() && strings.HasPrefix(ref.Name().String(), "refs/remotes/"+remote) {
			names = append(names, reg.ReplaceAllString(ref.Name().String(), ""))
		}
		return nil
	})
	return names, nil
}

// IsRemote returns true when passed a valid remote in the git repo
func IsRemote(remote string) (bool, error) {
	cmd := New("remote")
	cmd.Stdout = nil
	cmd.Stderr = nil
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

// Fetch a commit from a given remote
func Fetch(remote, commit string) error {
	gitcmd := []string{"fetch", remote, commit}
	cmd := New(gitcmd...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	if err != nil {
		return errors.Errorf("Can't fetch git commit %s from remote %s", commit, remote)
	}
	return nil
}

// Show all the commits between 2 git commits
func Show(commit1, commit2 string, reverse bool) {
	gitcmd := []string{"show"}
	if reverse {
		gitcmd = append(gitcmd, "--reverse")
	}
	gitcmd = append(gitcmd, fmt.Sprintf("%s..%s", commit1, commit2))
	New(gitcmd...).Run()
}

// GetLocalRemotes returns a string of local remote names and URLs
func GetLocalRemotes() (string, error) {
	cmd := New("remote", "-v")
	cmd.Stdout = nil
	remotes, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(remotes), nil
}

// GetLocalRemotesFromFile returns a string of local remote names and URLs based
// on their placement within .git/config file, which holds a different ordering
// compared to the alternatives presented by Remotes() and GetLocalRemotes().
func GetLocalRemotesFromFile() (string, error) {
	cmd := New("config", "--local", "--name-only", "--get-regex", "^remote.*")
	cmd.Stdout = nil
	remotes, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(remotes), nil
}

// NumberCommits returns the number of commits between two commit refs
func NumberCommits(sha1, sha2 string) int {
	cmd := New("log", "--oneline", fmt.Sprintf("%s..%s", sha1, sha2))
	cmd.Stdout = nil
	cmd.Stderr = nil
	CmdOut, err := cmd.Output()
	if err != nil {
		// silently fail and handle the return of 0 at caller
		return 0
	}
	numLines := strings.Count(string(CmdOut), "\n")
	return numLines
}
