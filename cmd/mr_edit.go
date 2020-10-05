package cmd

import (
	"fmt"
	"log"
	"runtime"
	"strconv"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrEditCmd = &cobra.Command{
	Use:     "edit [remote] <id>",
	Aliases: []string{"update"},
	Short:   "Edit or update an MR",
	Long:    ``,
	Example: `lab MR edit <id>                                # update MR via $EDITOR
lab MR update <id>                              # same as above
lab MR edit <id> -m "new title"                 # update title
lab MR edit <id> -m "new title" -m "new desc"   # update title & description
lab MR edit <id> -l newlabel --unlabel oldlabel # relabel MR`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rn, mrNum, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		mr, err := lab.MRGet(rn, int(mrNum))
		if err != nil {
			log.Fatal(err)
		}

		labels, err := cmd.Flags().GetStringSlice("label")
		if err != nil {
			log.Fatal(err)
		}

		// get the labels to remove
		unlabels, err := cmd.Flags().GetStringSlice("unlabel")
		if err != nil {
			log.Fatal(err)
		}

		labels, labelsChanged, err := editGetLabels(mr.Labels, labels, unlabels)
		if err != nil {
			log.Fatal(err)
		}

		// get the assignees to add
		assignees, err := cmd.Flags().GetStringSlice("assign")
		if err != nil {
			log.Fatal(err)
		}

		// get the assignees to remove
		unassignees, err := cmd.Flags().GetStringSlice("unassign")
		if err != nil {
			log.Fatal(err)
		}

		currentAssignees := mrGetCurrentAssignees(mr)
		assigneeIDs, assigneesChanged, err := getUpdateAssignees(currentAssignees, assignees, unassignees)
		if err != nil {
			log.Fatal(err)
		}

		// get all of the "message" flags
		msgs, err := cmd.Flags().GetStringSlice("message")
		if err != nil {
			log.Fatal(err)
		}
		title, body, err := editGetTitleDescription(mr.Title, mr.Description, msgs, cmd.Flags().NFlag())
		if err != nil {
			_, f, l, _ := runtime.Caller(0)
			log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
		}
		if title == "" {
			log.Fatal("aborting: empty mr title")
		}

		abortUpdate := title == mr.Title && body == mr.Description && !labelsChanged && !assigneesChanged
		if abortUpdate {
			log.Fatal("aborting: no changes")
		}

		linebreak, _ := cmd.Flags().GetBool("force-linebreak")
		if linebreak {
			body = textToMarkdown(body)
		}

		opts := &gitlab.UpdateMergeRequestOptions{
			Title:       &title,
			Description: &body,
		}

		if labelsChanged {
			opts.Labels = lab.Labels(labels)
		}

		if assigneesChanged {
			opts.AssigneeIDs = assigneeIDs
		}

		mrURL, err := lab.MRUpdate(rn, int(mrNum), opts)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(mrURL)
	},
}

// mrGetCurrentAssignees returns a string slice of the current assignees'
// usernames
func mrGetCurrentAssignees(mr *gitlab.MergeRequest) []string {
	currentAssignees := make([]string, len(mr.Assignees))
	if len(mr.Assignees) > 0 && mr.Assignees[0].Username != "" {
		for i, a := range mr.Assignees {
			currentAssignees[i] = a.Username
		}
	}
	return currentAssignees
}

func init() {
	mrEditCmd.Flags().StringSliceP("message", "m", []string{}, "Use the given <msg>; multiple -m are concatenated as separate paragraphs")
	mrEditCmd.Flags().StringSliceP("label", "l", []string{}, "Add the given label(s) to the merge request")
	mrEditCmd.Flags().StringSliceP("unlabel", "", []string{}, "Remove the given label(s) from the merge request")
	mrEditCmd.Flags().StringSliceP("assign", "a", []string{}, "Add an assignee by username")
	mrEditCmd.Flags().StringSliceP("unassign", "", []string{}, "Remove an assigne by username")
	mrEditCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks")

	mrCmd.AddCommand(mrEditCmd)
	carapace.Gen(mrEditCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
