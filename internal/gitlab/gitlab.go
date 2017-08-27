package gitlab

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/tcnksm/go-gitconfig"
	"github.com/xanzy/go-gitlab"
)

var (
	ErrProjectNotFound = errors.New("gitlab project not found")
)

var (
	lab   *gitlab.Client
	host  string
	token string
	User  string
)

func init() {
	var err error
	host, err = gitconfig.Entire("gitlab.host")
	if err != nil {
		log.Fatal("git config gitlab.host must be set")
	}
	token, err = gitconfig.Entire("gitlab.token")
	if err != nil {
		log.Fatal("git config gitlab.token must be set")
	}
	User, err = gitconfig.Entire("gitlab.user")
	if err != nil {
		log.Fatal("git config gitlab.user must be set")
	}

	if os.Getenv("DEBUG") != "" {
		log.Println("gitlab.host:", host)
		log.Println("gitlab.token:", "************"+token[12:])
		log.Println("gitlab.user:", User)
	}
	lab = gitlab.NewClient(nil, token)
	lab.SetBaseURL(host + "/api/v4")
	_, _, err = lab.Projects.ListProjects(&gitlab.ListProjectsOptions{})
	if err != nil {
		log.Fatal("Error: ", err)
	}
}

func FindProject(project string) (*gitlab.Project, error) {
	search := project
	// Assuming that a "/" in the project means its owned by an org
	if !strings.Contains(project, "/") {
		search = User + "/" + project
	}

	target, resp, err := lab.Projects.GetProject(search)
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrProjectNotFound
	}
	if err != nil {
		return nil, err
	}
	if os.Getenv("DEBUG") != "" {
		spew.Dump(target)
	}

	return target, nil
}

func ClonePath(project string) (string, error) {
	target, err := FindProject(project)
	if err != nil {
		return "", err
	}

	if target != nil {
		return target.SSHURLToRepo, nil
	}
	return project, nil
}

func Fork(project string) (string, error) {
	if !strings.Contains(project, "/") {
		return "", errors.New("remote must include namespace")
	}
	parts := strings.Split(project, "/")

	// See if a fork already exists
	target, err := FindProject(parts[1])
	if err == nil {
		return target.SSHURLToRepo, nil
	} else if err != nil && err != ErrProjectNotFound {
		return "", err
	}

	target, err = FindProject(project)
	if err != nil {
		return "", err
	}

	fork, _, err := lab.Projects.ForkProject(target.ID)
	if err != nil {
		return "", err
	}

	return fork.SSHURLToRepo, nil
}

func MergeRequest(project string, opts *gitlab.CreateMergeRequestOptions) (string, error) {
	if os.Getenv("DEBUG") != "" {
		spew.Dump(opts)
	}

	p, err := FindProject(project)
	if err != nil {
		return "", err
	}

	mr, _, err := lab.MergeRequests.CreateMergeRequest(p.ID, opts)
	if err != nil {
		return "", err
	}
	return mr.WebURL, nil
}
