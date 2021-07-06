package cmd

import (
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"
	"os"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrUnapproveCmd = &cobra.Command{
	Use:              "unapprove [remote] <id>",
	Aliases:          []string{},
	Short:            "Unapprove merge request",
	Example: heredoc.Doc(`
		lab mr unapprove origin
		lab mr unapprove upstream -F test_file
		lab mr unapprove upstream -m "A helpfull comment"
		lab mr unapprove upstream --with-comment
		lab mr unapprove upstream -m "A helpfull\nComment" --force-linebreak`),
	PersistentPreRun: labPersistentPreRun,
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

		msgs, err := cmd.Flags().GetStringArray("message")
		if err != nil {
			log.Fatal(err)
		}

		filename, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatal(err)
		}

		err = lab.MRUnapprove(p.ID, int(id))
		if err != nil {
			if err == lab.ErrStatusForbidden {
				log.Fatal(err)
			}
			if err == lab.ErrActionRepeated {
				fmt.Printf("Merge Request !%d already unapproved\n", id)
				os.Exit(1)
			}
		}

		if comment || len(msgs) > 0 || filename != "" {
			linebreak, err := cmd.Flags().GetBool("force-linebreak")
			if err != nil {
				log.Fatal(err)
			}

			createNote(rn, true, int(id), msgs, filename, linebreak)
		}

		fmt.Printf("Merge Request !%d unapproved\n", id)
	},
}

func init() {
	mrUnapproveCmd.Flags().Bool("with-comment", false, "Add a comment with the approval")
	mrUnapproveCmd.Flags().StringArrayP("message", "m", []string{}, "use the given <msg>; multiple -m are concatenated as separate paragraphs")
	mrUnapproveCmd.Flags().StringP("file", "F", "", "use the given file as the message")
	mrUnapproveCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks")
	mrCmd.AddCommand(mrUnapproveCmd)
	carapace.Gen(mrUnapproveCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
