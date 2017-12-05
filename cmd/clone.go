package cmd

import (
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/git"
	"github.com/zaquestion/lab/internal/gitlab"
)

// cloneCmd represents the clone command
// NOTE: There is special handling for "clone" in cmd/root.go
var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "",
	Long: `Clone supports these shorthands
- repo
- namespace/repo`,
	Run: func(cmd *cobra.Command, args []string) {
		project, err := gitlab.FindProject(args[0])
		if err == gitlab.ErrProjectNotFound {
			err = git.New(append([]string{"clone"}, args...)...).Run()
			if err != nil {
				log.Fatal(err)
			}
			return
		} else if err != nil {
			log.Fatal(err)
		}
		path := project.SSHURLToRepo
		if os.Getenv("DEBUG") != "" {
			log.Println("clonePath:", path)
		}
		err = git.New(append([]string{"clone", path}, args[1:]...)...).Run()
		if err != nil {
			log.Fatal(err)
		}

		// Clone project was a fork belonging to the user so user is
		// treating forks as origin. Add upstream as remoted pointing
		// to forked from repo
		if project.ForkedFromProject != nil &&
			strings.Contains(project.PathWithNamespace, gitlab.User()) {
			if len(args) > 1 {
				os.Chdir(args[1])
			} else {
				os.Chdir(project.Name)
			}
			ffProject, err := gitlab.FindProject(project.ForkedFromProject.PathWithNamespace)
			if err != nil {
				log.Fatal(err)
			}
			err = git.RemoteAdd("upstream", ffProject.SSHURLToRepo)
			if err != nil {
				log.Fatal(err)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(cloneCmd)
}
