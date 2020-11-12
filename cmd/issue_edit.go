package cmd

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var issueEditCmd = &cobra.Command{
	Use:     "edit [remote] <id>[:<comment_id>]",
	Aliases: []string{"update"},
	Short:   "Edit or update an issue",
	Long:    ``,
	Example: `lab issue edit <id>                                # update issue via $EDITOR
lab issue update <id>                              # same as above
lab issue edit <id> -m "new title"                 # update title
lab issue edit <id> -m "new title" -m "new desc"   # update title & description
lab issue edit <id> -l newlabel --unlabel oldlabel # relabel issue
lab issue edit <id>:<comment_id>                   # update a comment on MR`,
	Args:             cobra.MinimumNArgs(1),
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {

		rn, idString, err := parseArgsRemoteAndProject(args)
		if err != nil {
			log.Fatal(err)
		}

		var (
			issueNum   int = 0
			commentNum int = 0
		)

		if strings.Contains(idString, ":") {
			ids := strings.Split(idString, ":")
			issueNum, _ = strconv.Atoi(ids[0])
			commentNum, _ = strconv.Atoi(ids[1])
		} else {
			issueNum, _ = strconv.Atoi(idString)
		}

		issue, err := lab.IssueGet(rn, issueNum)
		if err != nil {
			log.Fatal(err)
		}

		linebreak, err := cmd.Flags().GetBool("force-linebreak")
		if err != nil {
			log.Fatal(err)
		}

		// Edit a comment on the Issue
		if commentNum != 0 {
			replyNote(rn, false, issueNum, commentNum, true, false, "", linebreak)
			return
		}

		// get the labels to add
		labels, err := cmd.Flags().GetStringSlice("label")
		if err != nil {
			log.Fatal(err)
		}

		// get the labels to remove
		unlabels, err := cmd.Flags().GetStringSlice("unlabel")
		if err != nil {
			log.Fatal(err)
		}

		labels, labelsChanged, err := editGetLabels(issue.Labels, labels, unlabels)
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

		currentAssignees := issueGetCurrentAssignees(issue)
		assigneeIDs, assigneesChanged, err := getUpdateAssignees(currentAssignees, assignees, unassignees)
		if err != nil {
			log.Fatal(err)
		}

		// get all of the "message" flags
		msgs, err := cmd.Flags().GetStringArray("message")
		if err != nil {
			log.Fatal(err)
		}
		title, body, err := editGetTitleDescription(issue.Title, issue.Description, msgs, cmd.Flags().NFlag())
		if err != nil {
			_, f, l, _ := runtime.Caller(0)
			log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
		}
		if title == "" {
			log.Fatal("aborting: empty issue title")
		}

		abortUpdate := title == issue.Title && body == issue.Description && !labelsChanged && !assigneesChanged
		if abortUpdate {
			log.Fatal("aborting: no changes")
		}

		if linebreak {
			body = textToMarkdown(body)
		}

		opts := &gitlab.UpdateIssueOptions{
			Title:       &title,
			Description: &body,
		}

		if labelsChanged {
			opts.Labels = labels
		}

		if assigneesChanged {
			opts.AssigneeIDs = assigneeIDs
		}

		issueURL, err := lab.IssueUpdate(rn, issueNum, opts)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(issueURL)
	},
}

// editGetLabels returns a string slice of labels based on the current
// labels and flags from the command line, and a bool indicating whether
// the labels have changed
func editGetLabels(idLabels []string, labels []string, unlabels []string) ([]string, bool, error) {
	// add the new labels to the current labels, then remove the "unlabels"
	labels = difference(union(idLabels, labels), unlabels)

	return labels, !same(idLabels, labels), nil
}

// issueGetCurrentAssignees returns a string slice of the current assignees'
// usernames
func issueGetCurrentAssignees(issue *gitlab.Issue) []string {
	currentAssignees := make([]string, len(issue.Assignees))
	if len(issue.Assignees) > 0 && issue.Assignees[0].Username != "" {
		for i, a := range issue.Assignees {
			currentAssignees[i] = a.Username
		}
	}
	return currentAssignees
}

// GetUpdateAssignees returns an int slice of assignee IDs based on the
// current assignees and flags from the command line, and a bool
// indicating whether the assignees have changed
func getUpdateAssignees(currentAssignees []string, assignees []string, unassignees []string) ([]int, bool, error) {
	// add the new assignees to the current assignees, then remove the "unassignees"
	assignees = difference(union(currentAssignees, assignees), unassignees)
	assigneesChanged := !same(currentAssignees, assignees)

	// turn the new assignee list into a list of assignee IDs
	var assigneeIDs []int
	if assigneesChanged && len(assignees) == 0 {
		// if we're removing all assignees, we have to use []int{0}
		// see https://github.com/xanzy/go-gitlab/issues/427
		assigneeIDs = []int{0}
	} else {
		assigneeIDs = make([]int, len(assignees))
		for i, a := range assignees {
			assigneeIDs[i] = *getAssigneeID(a)
		}
	}

	return assigneeIDs, assigneesChanged, nil
}

// editGetTitleDescription returns a title and description based on the current
// issue title and description and various flags from the command line
func editGetTitleDescription(title string, body string, msgs []string, nFlag int) (string, string, error) {
	if len(msgs) > 0 {
		title = msgs[0]

		if len(msgs) > 1 {
			body = strings.Join(msgs[1:], "\n\n")
		}

		// we have everything we need
		return title, body, nil
	}

	// if other flags were given (eg label), then skip the editor and return
	// what we already have
	if nFlag != 0 {
		return title, body, nil
	}

	text, err := editText(title, body)
	if err != nil {
		return "", "", err
	}
	return git.Edit("EDIT", text)
}

// editText returns an issue editing template that is suitable for loading
// into an editor
func editText(title string, body string) (string, error) {
	const tmpl = `{{.InitMsg}}

{{.CommentChar}} Edit the title and/or description. The first block of text
{{.CommentChar}} is the title and the rest is the description.`

	msg := &struct {
		InitMsg     string
		CommentChar string
	}{
		InitMsg:     title + "\n\n" + body,
		CommentChar: git.CommentChar(),
	}

	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	err = t.Execute(&b, msg)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

// union returns all the unique elements in a and b
func union(a, b []string) []string {
	mb := map[string]bool{}
	ab := []string{}
	for _, x := range b {
		mb[x] = true
		// add all of b's elements to ab
		ab = append(ab, x)
	}
	for _, x := range a {
		if _, ok := mb[x]; !ok {
			// if a's elements aren't in b, add them to ab
			// if they are, we don't need to add them
			ab = append(ab, x)
		}
	}
	return ab
}

// difference returns the elements in a that aren't in b
func difference(a, b []string) []string {
	mb := map[string]bool{}
	for _, x := range b {
		mb[x] = true
	}
	ab := []string{}
	for _, x := range a {
		if _, ok := mb[x]; !ok {
			ab = append(ab, x)
		}
	}
	return ab
}

// same returns true if a and b contain the same strings (regardless of order)
func same(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	mb := map[string]bool{}
	for _, x := range b {
		mb[x] = true
	}

	for _, x := range a {
		if _, ok := mb[x]; !ok {
			return false
		}
	}
	return true
}

func init() {
	issueEditCmd.Flags().StringArrayP("message", "m", []string{}, "use the given <msg>; multiple -m are concatenated as separate paragraphs")
	issueEditCmd.Flags().StringSliceP("label", "l", []string{}, "add the given label(s) to the issue")
	issueEditCmd.Flags().StringSliceP("unlabel", "", []string{}, "remove the given label(s) from the issue")
	issueEditCmd.Flags().StringSliceP("assign", "a", []string{}, "add an assignee by username")
	issueEditCmd.Flags().StringSliceP("unassign", "", []string{}, "remove an assignee by username")
	issueEditCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks")

	issueCmd.AddCommand(issueEditCmd)
	carapace.Gen(issueEditCmd).PositionalCompletion(
		action.Remotes(),
		action.Issues(issueList),
	)
}
