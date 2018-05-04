package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// projectCreateCmd represents the create command
var projectCreateCmd = &cobra.Command{
	Use:   "create [path]",
	Short: "Create a new project on GitLab",
	Long:  `If no path or name is provided the name of the git repo working directory`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var (
			name, _ = cmd.Flags().GetString("name")
			desc, _ = cmd.Flags().GetString("description")
		)
		path := determinePath(args, name)
		if path == "" && name == "" {
			log.Fatal("path or name must be set")
		}

		opts := gitlab.CreateProjectOptions{
			Path:        gitlab.String(path),
			Name:        gitlab.String(name),
			Description: gitlab.String(desc),
		}
		p, err := lab.ProjectCreate(&opts)
		if err != nil {
			log.Fatal(err)
		}
		if git.InsideGitRepo() {
			err = git.RemoteAdd("origin", p.SSHURLToRepo, ".")
			if err != nil {
				log.Fatal(err)
			}
		}
		fmt.Println(strings.TrimSuffix(p.HTTPURLToRepo, ".git"))
	},
}

func determinePath(args []string, name string) string {
	var path string
	if len(args) > 0 {
		path = args[0]
	}
	if path == "" && name == "" && git.InsideGitRepo() {
		wd, err := git.WorkingDir()
		if err != nil {
			log.Fatal(err)
		}
		p := strings.Split(wd, "/")
		path = p[len(p)-1]
	}
	return path
}

func init() {
	projectCreateCmd.Flags().StringP("name", "n", "", "name to use for the new project")
	projectCreateCmd.Flags().StringP("description", "d", "", "description to use for the new project")
	projectCmd.AddCommand(projectCreateCmd)
}
