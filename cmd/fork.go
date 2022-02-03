package cmd

import (
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/tcnksm/go-gitconfig"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	skipClone  = false
	waitFork   = true
	remoteName string
	targetData struct {
		project string
		group   string
		path    string
	}
	forkOpts *gitlab.ForkProjectOptions
)

// forkCmd represents the fork command
var forkCmd = &cobra.Command{
	Use:   "fork [remote|repo]",
	Short: "Fork a remote repository on GitLab and add as remote",
	Long: heredoc.Doc(`
		Fork a remote repository on user's location of choice.
		Both an already existent remote or a repository path can be specified.`),
	Example: heredoc.Doc(`
		lab fork origin
		lab fork upstream --remote-name origin
		lab fork origin --name new-awesome-project
		lab fork origin -g TheCoolestGroup -n InitialProject
		lab fork origin -p 'the_dot_git_path'
		lab fork origin -n 'new_fork' -r 'new_fork_remote'
		lab fork origin -s`),
	Args:             cobra.MaximumNArgs(1),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		skipClone, _ = cmd.Flags().GetBool("skip-clone")
		noWaitFork, _ := cmd.Flags().GetBool("no-wait")
		waitFork = !noWaitFork
		remoteName, _ = cmd.Flags().GetString("remote-name")
		targetData.project, _ = cmd.Flags().GetString("name")
		targetData.group, _ = cmd.Flags().GetString("group")
		targetData.path, _ = cmd.Flags().GetString("path")

		if targetData.project != "" || targetData.group != "" ||
			targetData.path != "" {
			forkOpts = &gitlab.ForkProjectOptions{
				Name:      gitlab.String(targetData.project),
				Namespace: gitlab.String(targetData.group),
				Path:      gitlab.String(targetData.path),
			}
		}

		remote, project := "", ""
		if len(args) == 1 {
			if ok, _ := git.IsRemote(args[0]); ok {
				remote = args[0]
			} else {
				project = args[0]
			}
		}

		if project != "" {
			forkCleanProject(project)
			return
		}

		if remote == "" {
			remote = "origin"
		}

		project, err := git.PathWithNamespace(remote)
		if err != nil {
			log.Fatal(err)
		}
		forkRemoteProject(project)
	},
}

// forkRemoteProject handle forks from within an already existent local
// repository (working directory), using git-remote information passed (or
// not) by the user. Since the directory already exists, only a new remote
// is added.
func forkRemoteProject(project string) {
	// Check for custom target namespace
	remote := determineForkRemote(project)
	if _, err := gitconfig.Local("remote." + remote + ".url"); err == nil {
		log.Fatalf("remote: %s already exists", remote)
	}

	forkRemoteURL, err := lab.Fork(project, forkOpts, useHTTP, waitFork)
	if err != nil {
		if err.Error() == "not finished" {
			fmt.Println("This fork is not ready yet and might take some minutes.")
		} else {
			log.Fatal(err)
		}
	}

	err = git.RemoteAdd(remote, forkRemoteURL, ".")
	if err != nil {
		log.Fatal(err)
	}
}

// forkCleanProject handle forks when the user passes a project name instead
// of a remote name directly. Usually it happens when the user is outside an
// existent local repository (working directory). Also, a clone step is
// performed when not explicitly skipped by the user with --skip-clone.
func forkCleanProject(project string) {
	// lab.Fork doesn't have access to the useHTTP var, so we need to pass
	// this info to that, so the process works correctly.
	_, err := lab.Fork(project, forkOpts, useHTTP, waitFork)
	if err != nil {
		if err.Error() == "not finished" && !skipClone {
			fmt.Println("This fork is not ready yet and might take some minutes.")
			skipClone = true
		} else {
			log.Fatal(err)
		}
	}

	if !skipClone {
		// the clone may happen in a different name/path when compared to
		// the original source project
		namespace := ""
		if targetData.group != "" {
			namespace = targetData.group + "/"
		}

		name := project
		if targetData.path != "" {
			name = targetData.path
		} else if targetData.project != "" {
			name = targetData.project
		} else {
			nameParts := strings.Split(name, "/")
			name = nameParts[len(nameParts)-1]
		}
		cloneCmd.Run(nil, []string{namespace + name})
	}
}

func determineForkRemote(project string) string {
	if remoteName != "" {
		return remoteName
	}

	name := lab.User()
	if targetData.group != "" {
		name = targetData.group
	}
	if strings.Split(project, "/")[0] == name {
		// #78 allow upstream remote to be added when "origin" is
		// referring to the user fork (and the fork already exists)
		name = "upstream"
	}
	return name
}

func init() {
	forkCmd.Flags().BoolP("skip-clone", "s", false, "skip clone after remote fork")
	forkCmd.Flags().Bool("no-wait", false, "don't wait for forking operation to finish")
	forkCmd.Flags().StringP("name", "n", "", "fork project with a different name")
	forkCmd.Flags().StringP("group", "g", "", "fork project in a different group (namespace)")
	forkCmd.Flags().StringP("path", "p", "", "fork project with a different path")
	forkCmd.Flags().StringP("remote-name", "r", "", "use a custom remote name for the fork")
	// useHTTP is defined in "util.go"
	forkCmd.Flags().BoolVar(&useHTTP, "http", false, "fork using HTTP protocol instead of SSH")
	RootCmd.AddCommand(forkCmd)
}
