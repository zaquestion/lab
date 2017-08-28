package cmd

import (
	"log"
	"os"

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
		path, err := gitlab.ClonePath(args[0])
		if err == gitlab.ErrProjectNotFound {
			git := git.New(append([]string{"clone"}, args...)...)
			err = git.Run()
			if err != nil {
				log.Fatal(err)
			}
		} else if err != nil {
			log.Fatal(err)
		}
		if os.Getenv("DEBUG") != "" {
			log.Println("clonePath:", path)
		}
		git := git.New(append([]string{"clone", path}, args[1:]...)...)
		err = git.Run()
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	RootCmd.AddCommand(cloneCmd)
}
