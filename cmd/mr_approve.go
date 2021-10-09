package cmd

import (
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
	"os"
)

var mrApproveCmd = &cobra.Command{
	Use:     "approve [remote] [<MR id or branch>]",
	Aliases: []string{},
	Short:   "Approve merge request",
	Example: heredoc.Doc(`
		lab mr approve origin
		lab mr approve upstream -F test_file
		lab mr approve upstream -m "A helpfull comment"
		lab mr approve upstream --with-comment
		lab mr approve upstream -m "A helpfull\nComment" --force-linebreak`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, id, err := parseArgsWithGitBranchMR(args)
		if err != nil {
			log.Fatal(err)
		}

		approvalConfig, err := lab.GetMRApprovalsConfiguration(rn, int(id))
		if err != nil {
			log.Fatal(err)
		}

		for _, approvers := range approvalConfig.ApprovedBy {
			if approvers.User.Username == lab.User() {
				fmt.Printf("Merge Request !%d already approved\n", id)
				os.Exit(1)
			}
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

		note := comment || len(msgs) > 0 || filename != ""
		linebreak := false
		if note {
			linebreak, err = cmd.Flags().GetBool("force-linebreak")
			if err != nil {
				log.Fatal(err)
			}
			if comment {
				state := noteGetState(rn, true, int(id))
				msg, _ := noteMsg(msgs, true, int(id), state, "", "")
				msgs = append(msgs, msg)
			}
		}

		msgs = append(msgs, "/approve")
		createNote(rn, true, int(id), msgs, filename, linebreak, "", note)

		fmt.Printf("Merge Request !%d approved\n", id)
	},
}

func init() {
	mrApproveCmd.Flags().Bool("with-comment", false, "Add a comment with the approval")
	mrApproveCmd.Flags().StringArrayP("message", "m", []string{}, "use the given <msg>; multiple -m are concatenated as separate paragraphs")
	mrApproveCmd.Flags().StringP("file", "F", "", "use the given file as the message")
	mrApproveCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks")
	mrCmd.AddCommand(mrApproveCmd)
	carapace.Gen(mrApproveCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
