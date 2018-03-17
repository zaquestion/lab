package cmd

import (
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tcnksm/go-gitconfig"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// forkCmd represents the fork command
var forkCmd = &cobra.Command{
	Use:   "fork [upstream-to-fork]",
	Short: "Fork a remote repository on GitLab and add as remote",
	Long:  ``,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			forkToUpstream(cmd, args)
			return
		}
		forkFromOrigin(cmd, args)
	},
}

func forkFromOrigin(cmd *cobra.Command, args []string) {
	if _, err := gitconfig.Local("remote." + lab.User() + ".url"); err == nil {
		log.Fatalf("remote: %s already exists", lab.User())
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
	forkRemoteURL, err := lab.Fork(project)
	if err != nil {
		log.Fatal(err)
	}

	name := determineForkRemote(project)
	err = git.RemoteAdd(name, forkRemoteURL, ".")
	if err != nil {
		log.Fatal(err)
	}
}
func forkToUpstream(cmd *cobra.Command, args []string) {
	_, err := lab.Fork(args[0])
	if err != nil {
		log.Fatal(err)
	}
	cloneCmd.Run(nil, []string{strings.Split(args[0], "/")[1]})
}
func determineForkRemote(project string) string {
	name := lab.User()
	if strings.Split(project, "/")[0] == lab.User() {
		// #78 allow upstream remote to be added when "origin" is
		// referring to the user fork (and the fork already exists)
		name = "upstream"
	}
	return name
}

func init() {
	RootCmd.AddCommand(forkCmd)
}
