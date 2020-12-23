package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tcnksm/go-gitconfig"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	skipClone  = false
	waitFork   = true
	targetData struct {
		project string
		group   string
		path    string
	}
	forkOpts *gitlab.ForkProjectOptions
)

// forkCmd represents the fork command
var forkCmd = &cobra.Command{
	Use:              "fork [upstream-to-fork]",
	Short:            "Fork a remote repository on GitLab and add as remote",
	Long:             ``,
	Args:             cobra.MaximumNArgs(1),
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		skipClone, _ = cmd.Flags().GetBool("skip-clone")
		noWaitFork, _ := cmd.Flags().GetBool("no-wait")
		waitFork = !noWaitFork
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

		if len(args) == 1 {
			forkToUpstream(cmd, args)
			return
		}
		forkFromOrigin(cmd, args)
	},
}

func forkFromOrigin(cmd *cobra.Command, args []string) {
	// Check for custom target namespace
	remote := lab.User()
	if targetData.group != "" {
		remote = targetData.group
	}

	if _, err := gitconfig.Local("remote." + remote + ".url"); err == nil {
		log.Fatalf("remote: %s already exists", remote)
	}
	if _, err := gitconfig.Local("remote.upstream.url"); err == nil {
		log.Fatal("remote: upstream already exists")
	}

	remoteURL, err := gitconfig.Local("remote.origin.url")
	if err != nil {
		log.Fatal(err)
	}
	if git.IsHub && strings.Contains(remoteURL, "github.com") {
		git := git.New("fork")
		git.Run()
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	project, err := git.PathWithNameSpace("origin")
	if err != nil {
		log.Fatal(err)
	}

	forkRemoteURL, err := lab.Fork(project, forkOpts, useHTTP, waitFork)
	if err != nil {
		if err.Error() == "not finished" {
			fmt.Println("This fork is not ready yet and might take some minutes.")
		} else {
			log.Fatal(err)
		}
	}

	name := determineForkRemote(project)
	err = git.RemoteAdd(name, forkRemoteURL, ".")
	if err != nil {
		log.Fatal(err)
	}
}

func forkToUpstream(cmd *cobra.Command, args []string) {
	project := args[0]
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
	// useHTTP is defined in "project_create.go"
	forkCmd.Flags().BoolVar(&useHTTP, "http", false, "fork using HTTP protocol instead of SSH")
	RootCmd.AddCommand(forkCmd)
}
