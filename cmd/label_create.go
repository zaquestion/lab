package cmd

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var labelCreateCmd = &cobra.Command{
	Use:     "create [remote] <name>",
	Aliases: []string{"add"},
	Short:   "Create a new label",
	Example: heredoc.Doc(`
		lab label create my-label
		lab label create --color cornflowerblue --description "Blue as a cornflower" blue
		lab label create --color #6495ed --description "Also blue as a cornflower" blue2`),
	PersistentPreRun: labPersistentPreRun,
	Args:             cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rn, name, err := parseArgsRemoteAndProject(args)
		if err != nil {
			log.Fatal(err)
		}

		color, err := cmd.Flags().GetString("color")
		if err != nil {
			log.Fatal(err)
		}

		desc, err := cmd.Flags().GetString("description")
		if err != nil {
			log.Fatal(err)
		}

		err = lab.LabelCreate(rn, &gitlab.CreateLabelOptions{
			Name:        &name,
			Description: &desc,
			Color:       &color,
		})

		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	labelCreateCmd.Flags().String("color", "#428BCA", "color of the new label in HTML hex notation or CSS color name")
	labelCreateCmd.Flags().String("description", "", "description of the new label")
	labelCmd.AddCommand(labelCreateCmd)
	carapace.Gen(labelCmd).PositionalCompletion(
		action.Remotes(),
	)
}
