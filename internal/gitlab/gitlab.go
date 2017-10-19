package gitlab

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/tcnksm/go-gitconfig"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
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

const defaultGitLabHost = "https://gitlab.com"

func init() {
	reader := bufio.NewReader(os.Stdin)
	var err error
	host, err = gitconfig.Entire("gitlab.host")
	if err != nil {
		fmt.Printf("Enter default GitLab host (default: %s): ", defaultGitLabHost)
		host, err = reader.ReadString('\n')
		host = host[:len(host)-1]
		if err != nil {
			log.Fatal(err)
		}
		if host == "" {
			host = defaultGitLabHost
		}
		cmd := git.New("config", "--global", "gitlab.host", host)
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}

	}
	var errt error
	User, err = gitconfig.Entire("gitlab.user")
	token, errt = gitconfig.Entire("gitlab.token")
	if err != nil {
		fmt.Print("Enter default GitLab user: ")
		User, err = reader.ReadString('\n')
		User = User[:len(User)-1]
		if err != nil {
			log.Fatal(err)
		}
		if User == "" {
			log.Fatal("git config gitlab.user must be set")
		}
		cmd := git.New("config", "--global", "gitlab.user", User)
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}

		// If the default user is being set this is the first time lab
		// is being run.
		if errt != nil {
			fmt.Print("Enter default GitLab token: ")
			token, err = reader.ReadString('\n')
			token = token[:len(token)-1]
			if err != nil {
				log.Fatal(err)
			}
			// Its okay for the key to be empty, since you can still call public repos
			if token != "" {
				cmd := git.New("config", "--global", "gitlab.token", token)
				err = cmd.Run()
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	lab = gitlab.NewClient(nil, token)
	lab.SetBaseURL(host + "/api/v4")

	if os.Getenv("DEBUG") != "" {
		log.Println("gitlab.host:", host)
		if len(token) > 12 {
			log.Println("gitlab.token:", "************"+token[12:])
		} else {
			log.Println("This token looks invalid due to it's length")
			log.Println("gitlab.token:", token)
		}
		log.Println("gitlab.user:", User)

		// Test listing projects
		projects, _, err := lab.Projects.ListProjects(&gitlab.ListProjectsOptions{})
		if err != nil {
			log.Fatal("Error: ", err)
		}
		if len(projects) > 0 {
			spew.Dump(projects[0])
		}
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

func ListMRs(project string, opts *gitlab.ListProjectMergeRequestsOptions) ([]*gitlab.MergeRequest, error) {
	p, err := FindProject(project)
	if err != nil {
		return nil, err
	}

	list, _, err := lab.MergeRequests.ListProjectMergeRequests(p.ID, opts)
	if err != nil {
		return nil, err
	}
	return list, nil
}
