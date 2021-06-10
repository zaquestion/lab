package cmd

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
)

var mrNoteCmd = &cobra.Command{
	Use:              "note [remote] <id>[:<comment_id>]",
	Aliases:          []string{"comment", "reply", "resolve"},
	Short:            "Add a note or comment to an MR on GitLab",
	PersistentPreRun: LabPersistentPreRun,
	Run:              NoteRunFn,
}

func init() {
	mrNoteCmd.Flags().StringArrayP("message", "m", []string{}, "use the given <msg>; multiple -m are concatenated as separate paragraphs")
	mrNoteCmd.Flags().StringP("file", "F", "", "use the given file as the message")
	mrNoteCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks")
	mrNoteCmd.Flags().Bool("quote", false, "quote note in reply (used with --reply only)")
	mrNoteCmd.Flags().Bool("resolve", false, "mark thread resolved (used with --reply only)")

	mrCmd.AddCommand(mrNoteCmd)
	carapace.Gen(mrNoteCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
