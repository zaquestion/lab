package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/pkg/errors"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrEditCmd = &cobra.Command{
	Use:     "edit [remote] <id>[:<comment_id>]",
	Aliases: []string{"update"},
	Short:   "Edit or update an MR",
	Example: heredoc.Doc(`
		lab mr edit 2
		lab mr edit 3 remote -m "new title"
		lab mr edit 5 upstream -m "new title" -m "new desc"
		lab mr edit 7 -l new_label --unlabel old_label
		lab mr edit 11 upstream -a johndoe -a janedoe 
		lab mr edit 17 upstream --unassign johndoe  
		lab mr edit 13 upstream --milestone "summer"
		lab mr edit 19 origin --target-branch other_branch
		lab mr edit 23 upstream -F test_file
		lab mr edit 29 upstream -F test_file --force-linebreak
		lab mr edit 31 upstream --draft
		lab mr edit 37 upstream --ready
		lab mr edit 41 upstream -r johndoe -r janedoe
		lab mr edit 43 upstream --unreview johndoe`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		commentNum, branchArgs, err := filterCommentArg(args)
		if err != nil {
			log.Fatal(err)
		}

		rn, id, err := parseArgsWithGitBranchMR(branchArgs)
		if err != nil {
			log.Fatal(err)
		}
		mrNum := int(id)

		if mrNum == 0 {
			fmt.Println("Error: Cannot determine MR id.")
			os.Exit(1)
		}

		mr, err := lab.MRGet(rn, mrNum)
		if err != nil {
			log.Fatal(err)
		}

		linebreak, err := cmd.Flags().GetBool("force-linebreak")
		if err != nil {
			log.Fatal(err)
		}

		// Edit a comment on the MR
		if commentNum != 0 {
			replyNote(rn, true, mrNum, commentNum, true, true, "", linebreak, false, nil)
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

		// get the reviewers to add
		reviewers, err := cmd.Flags().GetStringSlice("review")
		if err != nil {
			log.Fatal(err)
		}

		// get the reviewers to remove
		unreviewers, err := cmd.Flags().GetStringSlice("unreview")
		if err != nil {
			log.Fatal(err)
		}

		filename, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatal(err)
		}

		draft, err := cmd.Flags().GetBool("draft")
		if err != nil {
			log.Fatal(err)
		}

		ready, err := cmd.Flags().GetBool("ready")
		if err != nil {
			log.Fatal(err)
		}

		if draft && ready {
			log.Fatal("--draft and --ready cannot be used together")
		}

		currentAssignees := mrGetCurrentAssignees(mr)
		assigneeIDs, assigneesChanged, err := getUpdateUsers(currentAssignees, assignees, unassignees)
		if err != nil {
			log.Fatal(err)
		}

		currentReviewers := mrGetCurrentReviewers(mr)
		reviewerIDs, reviewersChanged, err := getUpdateUsers(currentReviewers, reviewers, unreviewers)
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

		targetBranchName, err := cmd.Flags().GetString("target-branch")
		if err != nil {
			log.Fatal(err)
		}

		targetBranchChanged := false
		if targetBranchName != "" {
			targetBranchName, err = getBranchName(rn, targetBranchName)
			if err != nil {
				log.Fatal(err)
			}

			if targetBranchName != mr.TargetBranch {
				targetBranchChanged = true
			}
		}

		// get all of the "message" flags
		msgs, err := cmd.Flags().GetStringSlice("message")
		if err != nil {
			log.Fatal(err)
		}

		var title, body string

		if filename != "" {
			if len(msgs) > 0 {
				log.Fatal("option -F cannot be combined with -m")
			}

			title, body, err = editGetTitleDescFromFile(filename)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			title, body, err = editGetTitleDescription(
				mr.Title, mr.Description, msgs, cmd.Flags().NFlag())
			if err != nil {
				_, f, l, _ := runtime.Caller(0)
				log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
			}
		}

		if title == "" {
			log.Fatal("aborting: empty mr title")
		}

		isWIP := hasPrefix(title, "wip:") ||
			hasPrefix(title, "[wip]")
		isDraft := hasPrefix(title, "draft:") ||
			hasPrefix(title, "[draft]") ||
			hasPrefix(title, "(draft)")

		if ready {
			if isWIP {
				if title[0] == '[' {
					title = strings.TrimPrefix(title, title[0:5])
				} else {
					title = strings.TrimPrefix(title, title[0:4])
				}
			} else if isDraft {
				if title[0] == '(' || title[0] == '[' {
					title = strings.TrimPrefix(title, title[0:7])
				} else {
					title = strings.TrimPrefix(title, title[0:6])
				}
			}
		}

		if draft {
			if isWIP {
				log.Fatal("the use of \"WIP\" terminology is deprecated, use \"Draft\" instead")
			}

			if !isDraft {
				title = "Draft: " + title
			}
		}

		abortUpdate := (title == mr.Title && body == mr.Description &&
			!labelsChanged && !assigneesChanged && !updateMilestone &&
			!targetBranchChanged && !reviewersChanged)
		if abortUpdate {
			log.Fatal("aborting: no changes")
		}

		if linebreak {
			body = textToMarkdown(body)
		}

		opts := &gitlab.UpdateMergeRequestOptions{
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

		if reviewersChanged {
			opts.ReviewerIDs = reviewerIDs
		}

		if updateMilestone {
			opts.MilestoneID = &milestoneID
		}

		if targetBranchChanged {
			opts.TargetBranch = &targetBranchName
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

// mrGetCurrentReviewers returns a string slice of the current reviewers'
// usernames
func mrGetCurrentReviewers(mr *gitlab.MergeRequest) []string {
	currentReviewers := make([]string, len(mr.Reviewers))
	if len(mr.Reviewers) > 0 && mr.Reviewers[0].Username != "" {
		for i, a := range mr.Reviewers {
			currentReviewers[i] = a.Username
		}
	}
	return currentReviewers
}

// getBranchName considers the possible ambiguity of different branch names
func getBranchName(project, branch string) (string, error) {
	opts := &gitlab.ListBranchesOptions{
		Search: &branch,
	}

	projectBranches, err := lab.BranchList(project, opts)
	if err != nil {
		return "", err
	}

	// Branch API accepts a search parameter, so we may get the answer
	// right away, however, the search term may match as a substring, so we
	// also need to check for multiple branch names and their ambiguity
	var match string

	switch len(projectBranches) {
	case 0:
		return "", errors.Errorf("Branch '%s' not found\n", branch)
	case 1:
		match = projectBranches[0].Name
	default:
		branchNames := make([]string, len(projectBranches))
		for _, branch := range projectBranches {
			branchNames = append(branchNames, branch.Name)
		}

		// Handle term ambiguity for multiple matched branch names
		matches, err := matchTerms([]string{branch}, branchNames)
		if err != nil {
			return "", errors.Errorf("Branch %s\n", err.Error())
		}

		// we only asked for a single term
		match = matches[0]
	}

	return match, nil
}

func init() {
	mrEditCmd.Flags().StringSliceP("message", "m", []string{}, "use the given <msg>; multiple -m are concatenated as separate paragraphs")
	mrEditCmd.Flags().StringSliceP("label", "l", []string{}, "add the given label(s) to the merge request")
	mrEditCmd.Flags().StringSliceP("unlabel", "", []string{}, "remove the given label(s) from the merge request")
	mrEditCmd.Flags().StringSliceP("assign", "a", []string{}, "add an assignee by username")
	mrEditCmd.Flags().StringSliceP("unassign", "", []string{}, "remove an assignee by username")
	mrEditCmd.Flags().String("milestone", "", "set milestone")
	mrEditCmd.Flags().StringP("target-branch", "t", "", "edit MR target branch")
	mrEditCmd.Flags().StringP("file", "F", "", "use the given file as the description")
	mrEditCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks")
	mrEditCmd.Flags().Bool("draft", false, "mark the merge request as draft")
	mrEditCmd.Flags().Bool("ready", false, "mark the merge request as ready")
	mrEditCmd.Flags().StringSliceP("review", "r", []string{}, "add an reviewer by username")
	mrEditCmd.Flags().StringSliceP("unreview", "", []string{}, "remove an reviewer by username")
	mrEditCmd.Flags().SortFlags = false

	mrCmd.AddCommand(mrEditCmd)

	carapace.Gen(mrEditCmd).FlagCompletion(carapace.ActionMap{
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

	carapace.Gen(mrEditCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
