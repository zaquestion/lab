package cmd

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
)

var issueNoteCmd = &cobra.Command{
	Use:     "note [remote] <id>[:<comment_id>]",
	Aliases: []string{"comment", "reply"},
	Short:   "Add a note or comment to an issue on GitLab",
	Args:    cobra.MinimumNArgs(1),
	Example: heredoc.Doc(`
		lab issue note 1
		lab issue note 2 -F test_file --force-linebreak
		lab issue note origin 2 -m "a message" -m "another one"
		lab issue note upstream 1:613278106 --quote`),
	PersistentPreRun: labPersistentPreRun,
	Run:              noteRunFn,
}

func init() {
	issueNoteCmd.Flags().StringArrayP("message", "m", []string{}, "use the given <msg>; multiple -m are concatenated as separate paragraphs")
	issueNoteCmd.Flags().StringP("file", "F", "", "use the given file as the message")
	issueNoteCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks")
	issueNoteCmd.Flags().Bool("quote", false, "quote note in reply")
	issueNoteCmd.Flags().Bool("resolve", false, "[unused in issue note command]")
	issueNoteCmd.Flags().MarkHidden("resolve")
	issueNoteCmd.Flags().StringP("commit", "", "", "[unused in issue note command]")
	issueNoteCmd.Flags().MarkHidden("commit")

	issueCmd.AddCommand(issueNoteCmd)
	carapace.Gen(issueNoteCmd).PositionalCompletion(
		action.Remotes(),
		action.Issues(issueList),
	)
}
