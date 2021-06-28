package cmd

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
)

var issueNoteCmd = &cobra.Command{
	Use:              "note [remote] <id>[:<comment_id>]",
	Aliases:          []string{"comment", "reply"},
	Short:            "Add a note or comment to an issue on GitLab",
	Args:             cobra.MinimumNArgs(1),
	PersistentPreRun: labPersistentPreRun,
	Run:              noteRunFn,
}

func init() {
	issueNoteCmd.Flags().StringArrayP("message", "m", []string{}, "use the given <msg>; multiple -m are concatenated as separate paragraphs")
	issueNoteCmd.Flags().StringP("file", "F", "", "use the given file as the message")
	issueNoteCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks")
	issueNoteCmd.Flags().Bool("quote", false, "quote note in reply (used with --reply only)")
	issueNoteCmd.Flags().Bool("resolve", false, "[unused in issue note command]")

	issueCmd.AddCommand(issueNoteCmd)
	carapace.Gen(issueNoteCmd).PositionalCompletion(
		action.Remotes(),
		action.Issues(issueList),
	)
}
