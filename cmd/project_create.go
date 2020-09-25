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

// private and public are defined in snippet_create.go
var internal bool

// projectCreateCmd represents the create command
var projectCreateCmd = &cobra.Command{
	Use:   "create [path]",
	Short: "Create a new project on GitLab",
	Long: `Create a new project on GitLab in your user namespace.

path refers to the path on GitLab not including the group/namespace. If no path
or name is provided and the current directory is a git repo, the name of the
current working directory will be used.`,
	Example: `# this command...                          # creates this project
lab project create                         # user/<curr dir> named <curr dir>
                                           # (above only works w/i git repo)
lab project create myproject               # user/myproject named myproject
lab project create myproject -n "new proj" # user/myproject named "new proj"
lab project create -n "new proj"           # user/new-proj named "new proj"`,
	Args:             cobra.MaximumNArgs(1),
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			name, _ = cmd.Flags().GetString("name")
			desc, _ = cmd.Flags().GetString("description")
		)
		path := determinePath(args, name)
		if path == "" && name == "" {
			log.Fatal("path or name must be set")
		}

		// set the default visibility
		visibility := gitlab.PrivateVisibility

		// now override the visibility if the user passed in relevant flags. if
		// the user passes multiple flags, this will use the "most private"
		// option given, ignoring the rest
		switch {
		case private:
			visibility = gitlab.PrivateVisibility
		case internal:
			visibility = gitlab.InternalVisibility
		case public:
			visibility = gitlab.PublicVisibility
		}

		opts := gitlab.CreateProjectOptions{
			Path:                 gitlab.String(path),
			Name:                 gitlab.String(name),
			Description:          gitlab.String(desc),
			Visibility:           &visibility,
			ApprovalsBeforeMerge: gitlab.Int(0),
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
	projectCreateCmd.Flags().StringP("name", "n", "", "name of the new project")
	projectCreateCmd.Flags().StringP("description", "d", "", "description of the new project")
	projectCreateCmd.Flags().BoolVarP(&private, "private", "p", false, "Make project private: visible only to project members")
	projectCreateCmd.Flags().BoolVar(&public, "public", false, "Make project public: visible without any authentication")
	projectCreateCmd.Flags().BoolVar(&internal, "internal", false, "Make project internal: visible to any authenticated user (default)")
	projectCmd.AddCommand(projectCreateCmd)
}
