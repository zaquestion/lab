package cmd

import (
	"fmt"
	"log"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrUnapproveCmd = &cobra.Command{
	Use:              "unapprove [remote] <id>",
	Aliases:          []string{},
	Short:            "Unapprove merge request",
	Long:             ``,
	Args:             cobra.MinimumNArgs(1),
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsWithGitBranchMR(args)
		if err != nil {
			log.Fatal(err)
		}

		p, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
		}

		comment, err := cmd.Flags().GetBool("with-comment")
		if err != nil {
			log.Fatal(err)
		}
		if comment {
			msgs, err := cmd.Flags().GetStringArray("message")
			if err != nil {
				log.Fatal(err)
			}

			filename, err := cmd.Flags().GetString("file")
			if err != nil {
				log.Fatal(err)
			}

			linebreak, err := cmd.Flags().GetBool("force-linebreak")
			if err != nil {
				log.Fatal(err)
			}

			createNote(rn, true, int(id), msgs, filename, linebreak)
		}

		err = lab.MRUnapprove(p.ID, int(id))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Merge Request #%d Unapproved\n", id)
	},
}

func init() {
	mrUnapproveCmd.Flags().Bool("with-comment", false, "Add a comment with the approval")
	mrUnapproveCmd.Flags().StringArrayP("message", "m", []string{}, "use the given <msg>; multiple -m are concatenated as separate paragraphs (used with --with-comment only)")
	mrUnapproveCmd.Flags().StringP("file", "F", "", "use the given file as the message (used with --with-comment only)")
	mrUnapproveCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks (used with --with-comment only)")
	mrCmd.AddCommand(mrUnapproveCmd)
	carapace.Gen(mrUnapproveCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
