package cmd

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
)

var mrNoteCmd = &cobra.Command{
	Use:              "note [remote] <id>",
	Aliases:          []string{"comment"},
	Short:            "Add a note or comment to an MR on GitLab",
	Long:             ``,
	Args:             cobra.MinimumNArgs(1),
	PersistentPreRun: LabPersistentPreRun,
	Run:              NoteRunFn,
}

func init() {
	mrNoteCmd.Flags().StringArrayP("message", "m", []string{}, "Use the given <msg>; multiple -m are concatenated as separate paragraphs")
	mrNoteCmd.Flags().StringP("file", "F", "", "Use the given file as the message")
	mrNoteCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks")
	mrCmd.AddCommand(mrNoteCmd)
	carapace.Gen(mrNoteCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
