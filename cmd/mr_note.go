package cmd

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
)

var mrNoteCmd = &cobra.Command{
	Use:              "note [remote] <id>[:<comment_id>]",
	Aliases:          []string{"comment", "reply", "resolve"},
	Short:            "Add a note or comment to an MR on GitLab",
	Example: heredoc.Doc(`
		lab mr note origin
		lab mr note upstream -F test_file
		lab mr note a_remote -F test_file --force-linebreak
		lab mr note upstream -m "A helpfull comment"
		lab mr note upstream:613278106 --quote
		lab mr note upstream:613278107 --resolve`),
	PersistentPreRun: labPersistentPreRun,
	Run:              noteRunFn,
}

func init() {
	mrNoteCmd.Flags().StringArrayP("message", "m", []string{}, "use the given <msg>; multiple -m are concatenated as separate paragraphs")
	mrNoteCmd.Flags().StringP("file", "F", "", "use the given file as the message")
	mrNoteCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks")
	mrNoteCmd.Flags().Bool("quote", false, "quote note in reply")
	mrNoteCmd.Flags().Bool("resolve", false, "mark thread resolved")

	mrCmd.AddCommand(mrNoteCmd)
	carapace.Gen(mrNoteCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
