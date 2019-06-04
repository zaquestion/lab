package cmd

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
	zsh "github.com/rsteube/cobra-zsh-gen"
)

var issueEditCmd = &cobra.Command{
	Use:     "edit [remote] <id>",
	Aliases: []string{"update"},
	Short:   "Edit or update an issue",
	Long:    ``,
	Example: `lab issue edit <id>                                # update issue via $EDITOR
lab issue update <id>                              # same as above
lab issue edit <id> -m "new title"                 # update title
lab issue edit <id> -m "new title" -m "new desc"   # update title & description
lab issue edit <id> -l newlabel --unlabel oldlabel # relabel issue`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// get remote and issue from cmd arguments
		rn, issueNum, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		// get existing issue
		issue, err := lab.IssueGet(rn, int(issueNum))
		if err != nil {
			log.Fatal(err)
		}

		labels, labelsChanged, err := issueEditGetLabels(issue, cmd.Flags())
		if err != nil {
			log.Fatal(err)
		}

		assigneeIDs, assigneesChanged, err := issueEditGetAssignees(issue, cmd.Flags())
		if err != nil {
			log.Fatal(err)
		}

		title, body, err := issueEditGetTitleDescription(issue, cmd.Flags())
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

		opts := &gitlab.UpdateIssueOptions{
			Title:       &title,
			Description: &body,
		}

		if labelsChanged {
			opts.Labels = gitlab.Labels(labels)
		}

		if assigneesChanged {
			opts.AssigneeIDs = assigneeIDs
		}

		issueURL, err := lab.IssueUpdate(rn, int(issueNum), opts)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(issueURL)
	},
}

// issueEditGetLabels returns a string slice of issues based on the current
// issue labels and flags from the command line, and a bool indicating whether
// the labels have changed
func issueEditGetLabels(issue *gitlab.Issue, flags *pflag.FlagSet) ([]string, bool, error) {
	// get the labels to add
	labels, err := flags.GetStringSlice("label")
	if err != nil {
		return []string{}, false, err
	}

	// get the labels to remove
	unlabels, err := flags.GetStringSlice("unlabel")
	if err != nil {
		return []string{}, false, err
	}

	// add the new labels to the current labels, then remove the "unlabels"
	labels = difference(union(issue.Labels, labels), unlabels)

	return labels, !same(issue.Labels, labels), nil
}

// issueEditGetAssignees returns an int slice of assignee IDs based on the
// current issue assignees and flags from the command line, and a bool
// indicating whether the assignees have changed
func issueEditGetAssignees(issue *gitlab.Issue, flags *pflag.FlagSet) ([]int, bool, error) {
	currentAssignees := make([]string, len(issue.Assignees))
	if len(issue.Assignees) > 0 && issue.Assignees[0].Username != "" {
		for i, a := range issue.Assignees {
			currentAssignees[i] = a.Username
		}
	}

	// get the assignees to add
	assignees, err := flags.GetStringSlice("assign")
	if err != nil {
		return []int{}, false, err
	}

	// get the assignees to remove
	unassignees, err := flags.GetStringSlice("unassign")
	if err != nil {
		return []int{}, false, err
	}

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

// issueEditGetTitleDescription returns a title and description for an issue
// based on the current issue title and description and various flags from the
// command line
func issueEditGetTitleDescription(issue *gitlab.Issue, flags *pflag.FlagSet) (string, string, error) {
	title, body := issue.Title, issue.Description

	// get all of the "message" flags
	msgs, err := flags.GetStringSlice("message")
	if err != nil {
		log.Fatal(err)
	}

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
	if flags.NFlag() != 0 {
		return title, body, nil
	}

	text, err := issueEditText(title, body)
	if err != nil {
		return "", "", err
	}
	return git.Edit("ISSUE_EDIT", text)
}

// issueEditText returns an issue editing template that is suitable for loading
// into an editor
func issueEditText(title string, body string) (string, error) {
	const tmpl = `{{.InitMsg}}

{{.CommentChar}} Edit the title and/or description of this issue. The first
{{.CommentChar}} block of text is the title and the rest is the description.`

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

// issueEditCmdAddFlags adds various flags to the `lab issue edit` command
func issueEditCmdAddFlags(flags *pflag.FlagSet) *pflag.FlagSet {
	flags.StringSliceP("message", "m", []string{}, "Use the given <msg>; multiple -m are concatenated as separate paragraphs")
	flags.StringSliceP("label", "l", []string{}, "Add the given label(s) to the issue")
	flags.StringSliceP("unlabel", "", []string{}, "Remove the given label(s) from the issue")
	flags.StringSliceP("assign", "a", []string{}, "Add an assignee by username")
	flags.StringSliceP("unassign", "", []string{}, "Remove an assigne by username")
	return flags
}

func init() {
	issueEditCmdAddFlags(issueEditCmd.Flags())
	zsh.Wrap(issueEditCmd).MarkZshCompPositionalArgumentCustom(1, "__lab_completion_remote")
	zsh.Wrap(issueEditCmd).MarkZshCompPositionalArgumentCustom(2, "__lab_completion_issue $words[2]")
	issueCmd.AddCommand(issueEditCmd)
}
