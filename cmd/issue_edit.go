package cmd

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/MakeNowJust/heredoc/v2"
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
	Example: heredoc.Doc(`
		lab issue edit 14
		lab issue edit 14:2065489
		lab issue edit 14 -a johndoe --unassign jackdoe
		lab issue edit 14 -m "new title"
		lab issue edit 14 -m "new title" -m "new desc"
		lab issue edit 14 -l new_label --unlabel old_label
		lab issue edit --milestone "NewYear"
		lab issue edit --force-linebreak`),
	Args:             cobra.MinimumNArgs(1),
	PersistentPreRun: labPersistentPreRun,
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
			replyNote(rn, false, issueNum, commentNum, true, false, "", linebreak, false, nil)
			return
		}

		var labelsChanged bool
		// get the labels to add
		addLabelTerms, err := cmd.Flags().GetStringSlice("label")
		if err != nil {
			log.Fatal(err)
		}
		addLabels, err := mapLabels(rn, addLabelTerms)
		if err != nil {
			log.Fatal(err)
		}
		if len(addLabels) > 0 {
			labelsChanged = true
		}

		// get the labels to remove
		rmLabelTerms, err := cmd.Flags().GetStringSlice("unlabel")
		if err != nil {
			log.Fatal(err)
		}
		rmLabels, err := mapLabels(rn, rmLabelTerms)
		if err != nil {
			log.Fatal(err)
		}
		if len(rmLabels) > 0 {
			labelsChanged = true
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
		assigneeIDs, assigneesChanged, err := getUpdateUsers(currentAssignees, assignees, unassignees)
		if err != nil {
			log.Fatal(err)
		}

		milestoneName, err := cmd.Flags().GetString("milestone")
		if err != nil {
			log.Fatal(err)
		}
		updateMilestone := cmd.Flags().Lookup("milestone").Changed
		milestoneID := -1

		if milestoneName != "" {
			ms, err := lab.MilestoneGet(rn, milestoneName)
			if err != nil {
				log.Fatal(err)
			}
			milestoneID = ms.ID
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

		abortUpdate := title == issue.Title && body == issue.Description && !labelsChanged && !assigneesChanged && !updateMilestone
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
			// empty arrays are just ignored
			opts.AddLabels = addLabels
			opts.RemoveLabels = rmLabels
		}

		if assigneesChanged {
			opts.AssigneeIDs = assigneeIDs
		}

		if updateMilestone {
			opts.MilestoneID = &milestoneID
		}

		issueURL, err := lab.IssueUpdate(rn, issueNum, opts)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(issueURL)
	},
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

// editText returns an issue editing template that is suitable for loading
// into an editor
func editText(title string, body string) (string, error) {
	tmpl := heredoc.Doc(`
		{{.InitMsg}}

		{{.CommentChar}} Edit the title and/or description. The first block of text
		{{.CommentChar}} is the title and the rest is the description.`)

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

func init() {
	issueEditCmd.Flags().StringArrayP("message", "m", []string{}, "use the given <msg>; multiple -m are concatenated as separate paragraphs")
	issueEditCmd.Flags().StringSliceP("label", "l", []string{}, "add the given label(s) to the issue")
	issueEditCmd.Flags().StringSliceP("unlabel", "", []string{}, "remove the given label(s) from the issue")
	issueEditCmd.Flags().StringSliceP("assign", "a", []string{}, "add an assignee by username")
	issueEditCmd.Flags().StringSliceP("unassign", "", []string{}, "remove an assignee by username")
	issueEditCmd.Flags().String("milestone", "", "set milestone")
	issueEditCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks")
	issueEditCmd.Flags().SortFlags = false

	issueCmd.AddCommand(issueEditCmd)

	carapace.Gen(issueEditCmd).FlagCompletion(carapace.ActionMap{
		"label": carapace.ActionMultiParts(",", func(c carapace.Context) carapace.Action {
			project, _, err := parseArgsRemoteAndProject(c.Args)
			if err != nil {
				return carapace.ActionMessage(err.Error())
			}
			return action.Labels(project).Invoke(c).Filter(c.Parts).ToA()
		}),
		"milestone": carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			project, _, err := parseArgsRemoteAndProject(c.Args)
			if err != nil {
				return carapace.ActionMessage(err.Error())
			}
			return action.Milestones(project, action.MilestoneOpts{Active: true})
		}),
	})

	carapace.Gen(issueEditCmd).PositionalCompletion(
		action.Remotes(),
		action.Issues(issueList),
	)
}
