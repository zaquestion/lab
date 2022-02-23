package cmd

import (
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	retry "github.com/avast/retry-go"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/git"
	"github.com/zaquestion/lab/internal/gitlab"
)

// cloneCmd represents the clone command
var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "GitLab aware clone repo command",
	Long: heredoc.Doc(`
		Clone a repository, similarly to 'git clone', but aware of GitLab
		specific settings.`),
	Example: heredoc.Doc(`
		lab clone awesome-repo
		lab clone company/awesome-repo --http
		lab clone company/backend-team/awesome-repo`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatal("You must specify a repository to clone.")
		}

		useHTTP, err := cmd.Flags().GetBool("http")
		if err != nil {
			log.Fatal(err)
		}

		if useHTTP {
			args = append(args, []string{"--http"}...)
		}

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
		path := labURLToRepo(project)
		// #116 retry on the cases where we found a project but clone
		// failed over ssh
		err = retry.Do(func() error {
			return git.New(append([]string{"clone", path}, args[1:]...)...).Run()
		}, retry.Attempts(3), retry.Delay(time.Second))
		if err != nil {
			log.Fatal(err)
		}

		// Clone project was a fork belonging to the user; user is
		// treating forks as origin. Add upstream as remoted pointing
		// to forked from repo
		if project.ForkedFromProject != nil &&
			strings.Contains(project.PathWithNamespace, gitlab.User()) {
			var dir string
			if len(args) > 1 {
				dir = args[1]
			} else {
				dir = project.Path
			}
			ffProject, err := gitlab.FindProject(project.ForkedFromProject.PathWithNamespace)
			if err != nil {
				log.Fatal(err)
			}
			urlToRepo := labURLToRepo(ffProject)
			err = git.RemoteAdd("upstream", urlToRepo, "./"+dir)
			if err != nil {
				log.Fatal(err)
			}
		}
	},
}

func init() {
	// useHTTP is defined in "project_create.go"
	cloneCmd.Flags().BoolVar(&useHTTP, "http", false, "clone using HTTP protocol instead of SSH")
	RootCmd.AddCommand(cloneCmd)
}
