package git

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	retry "github.com/avast/retry-go"
	"github.com/pkg/errors"
	gitconfig "github.com/tcnksm/go-gitconfig"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

var (
	// IsHub is true when using "hub" as the git binary
	IsHub bool
	repo  *git.Repository
)

func init() {
	_, err := exec.LookPath("hub")
	if err == nil {
		IsHub = true
	}
	repo, err = git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		log.Fatal(err)
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
	wd, err := WorkingDir()
	if err != nil {
		return "", err
	}
	return path.Join(wd, git.GitDirName), nil
}

// WorkingDir returns the full pall to the root of the current git repository
func WorkingDir() (string, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return "", err
	}
	return wt.Filesystem.Root(), nil
}

// CommentChar returns active comment char and defaults to '#'
func CommentChar() string {
	char, err := gitconfig.Entire("core.commentChar")
	if err == nil {
		return char
	}
	return "#"
}

// LastCommitMessage returns the last commits message as one line
func LastCommitMessage() (string, error) {
	log, err := repo.Log(&git.LogOptions{})
	if err != nil {
		return "", err
	}
	commit, err := log.Next()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(fmt.Sprintf("%s", commit.Message)), nil
}

// Log produces a formatted gitlog between 2 git shas
func Log(sha1, sha2 string) (string, error) {
	s1, err := repo.ResolveRevision(plumbing.Revision(sha1))
	if err != nil {
		return "", err
	}
	s2, err := repo.ResolveRevision(plumbing.Revision(sha2))
	if err != nil {
		return "", err
	}
	log, err := repo.Log(&git.LogOptions{From: *s2})
	if err != nil {
		return "", err
	}
	var s strings.Builder
	err = log.ForEach(func(c *object.Commit) error {
		if c.Hash.String() == s1.String() {
			return storer.ErrStop
		}
		s.WriteString(fmt.Sprintf("%s (%s, %s)\n%s\n", c.Hash.String()[:7], c.Author.Name, ago(time.Now().Sub(c.Author.When)), indent(wrap(c.Message))))
		return nil
	})
	if err != nil {
		return "", err
	}

	return s.String(), nil
}

func wrap(s string) string {
	var output []string
	for _, line := range strings.Split(s, "\n") {
		if len(line) > 78 {
			line = line[:78] + "\n" + line[78:]
		}

		output = append(output, line)
	}

	return strings.Join(output, "\n")
}

func indent(s string) string {
	var output []string
	for _, line := range strings.Split(s, "\n") {
		if len(line) != 0 {
			line = "   " + line
		}

		output = append(output, line)
	}

	return strings.Join(output, "\n")
}

func ago(d time.Duration) string {
	const (
		day   = time.Hour * 24
		month = day * 30
		year  = day * 365
	)
	d = d.Round(time.Second)
	Y := d / year
	if Y > 1 {
		return fmt.Sprintf("%d years ago", Y)
	}
	if Y == 1 {
		return fmt.Sprint("1 year ago")
	}

	M := d / month
	if M > 1 {
		return fmt.Sprintf("%d months ago", M)
	}
	if M == 1 {
		return fmt.Sprint("1 month ago")
	}

	D := d / day
	if D > 1 {
		return fmt.Sprintf("%d days ago", D)
	}
	if D == 1 {
		return fmt.Sprint("1 day ago")
	}

	h := d / time.Hour
	if h > 1 {
		return fmt.Sprintf("%d hours ago", h)
	}
	if h == 1 {
		return fmt.Sprint("1 hour ago")
	}

	m := d / time.Minute
	if m > 1 {
		return fmt.Sprintf("%d minutes ago", m)
	}
	if m == 1 {
		return fmt.Sprint("1 minute ago")
	}

	s := d / time.Second
	if s > 1 {
		return fmt.Sprintf("%d seconds ago", s)
	}
	if s == 1 {
		return fmt.Sprint("1 second ago")
	}
	return fmt.Sprint("0 seconds ago")
}

// CurrentBranch returns the currently checked out branch and strips away all
// but the branchname itself.
func CurrentBranch() (string, error) {
	head, err := repo.Head()
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(head.Name().String(), "/", 3)
	if len(parts) < 3 {
		return "", errors.Errorf("Could not parse branch from ref: %s", head.Name())
	}
	return parts[2], nil
}

// PathWithNameSpace returns the owner/repository for the current repo
// Such as zaquestion/lab
// Respects GitLab subgroups (https://docs.gitlab.com/ce/user/group/subgroups/)
func PathWithNameSpace(remote string) (string, error) {
	config, err := repo.Config()
	if err != nil {
		return "", err
	}

	rc, ok := config.Remotes[remote]
	if !ok {
		return "", errors.Errorf("remote %s could not be found", remote)
	}

	remoteURL := rc.URLs[0]
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
	remote, err := repo.CreateRemote(&config.RemoteConfig{
		Name: name,
		URLs: []string{url},
	})
	if err != nil {
		return err
	}
	fmt.Println("Updating", name)

	err = retry.Do(func() error {
		return remote.Fetch(&git.FetchOptions{
			Progress: os.Stdout,
		})
	}, retry.Attempts(3), retry.Delay(time.Second), retry.Units(time.Nanosecond))
	if err != nil {
		return err
	}
	fmt.Println("new remote:", name)
	return nil
}

// IsRemote returns true when passed a valid remote in the git repo
func IsRemote(remote string) (bool, error) {
	config, err := repo.Config()
	if err != nil {
		return false, err
	}

	_, ok := config.Remotes[remote]
	return ok, nil
}

// InsideGitRepo returns true when the current working directory is inside the
// working tree of a git repo
func InsideGitRepo() bool {
	_, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
	return err != git.ErrRepositoryNotExists
}
