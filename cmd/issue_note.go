package cmd

import (
	"fmt"
	"os"

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

func noteRunFn(cmd *cobra.Command, args []string) {
	isMR := false
	if os.Args[1] == "mr" {
		isMR = true
	}

	reply, branchArgs, err := filterCommentArg(args)
	if err != nil {
		log.Fatal(err)
	}

	var (
		rn    string
		idNum int = 0
	)

	if isMR {
		s, mrNum, _ := parseArgsWithGitBranchMR(branchArgs)
		if mrNum == 0 {
			fmt.Println("Error: Cannot determine MR id.")
			os.Exit(1)
		}
		idNum = int(mrNum)
		rn = s
	} else {
		s, issueNum, _ := parseArgsRemoteAndID(branchArgs)
		if issueNum == 0 {
			fmt.Println("Error: Cannot determine issue id.")
			os.Exit(1)
		}
		idNum = int(issueNum)
		rn = s
	}

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

	if reply != 0 {
		resolve, err := cmd.Flags().GetBool("resolve")
		if err != nil {
			log.Fatal(err)
		}
		// 'lab mr resolve' always overrides options
		if os.Args[2] == "resolve" {
			resolve = true
		}

		quote, err := cmd.Flags().GetBool("quote")
		if err != nil {
			log.Fatal(err)
		}
		replyNote(rn, isMR, int(idNum), reply, quote, false, filename, linebreak, resolve, msgs)
		return
	}

	createNote(rn, isMR, int(idNum), msgs, filename, linebreak)
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
