package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/fatih/color"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"github.com/pkg/errors"
	"github.com/rsteube/carapace"
	"github.com/savioxavier/termlink"
	"github.com/spf13/cobra"
	"github.com/zaquestion/lab/internal/action"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
	"golang.org/x/term"
)

var (
	mrLabels       []string
	mrState        string
	mrTargetBranch string
	mrMilestone    string
	mrNumRet       string
	mrAll          bool
	mrMine         bool
	mrAuthor       string
	mrAuthorID     *int
	mrDraft        bool
	mrReady        bool
	mrConflicts    bool
	mrNoConflicts  bool
	mrExactMatch   bool
	mrApprover     string
	mrApproverID   *gitlab.ApproverIDsValue
	mrAssignee     string
	mrAssigneeID   *gitlab.AssigneeIDValue
	mrOrder        string
	mrSortedBy     string
	mrReviewer     string
	mrReviewerID   *gitlab.ReviewerIDValue
)

func truncateText(s string, length int) (string) {
	if length > len(s) {
		return s
	}
	return s[:length]
}

func printRED(text string, cols int) {
	fmt.Printf(color.RedString("%-"+fmt.Sprintf("%d", cols)+"s", text))
}

func printGREEN(text string, cols int) {
	fmt.Printf(color.GreenString("%-"+fmt.Sprintf("%d", cols)+"s", text))
}

func printYELLOW(text string, cols int) {
	fmt.Printf(color.YellowString("%-"+fmt.Sprintf("%d", cols)+"s", text))
}

func printColumns(data [][]string) {
	spacing := 2 // adjust this variable to increase/decrease spacing between columns

	columnWidths := make([]int, len(data[0]))
	for _, row := range data {
		for cellnum, cell := range row {
			if cellnum == 0 { // the 0th entry is the webURL and is not output
				continue
			}
			if len(cell) > columnWidths[cellnum] {
				columnWidths[cellnum] = len(cell)
			}
		}
	}

	// Make sure the output fits on the screen.  If it does not, then truncate
	// the title field.

	// get the screen resolution (only width is needed)
	width, _, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatal(err)
	}

	// Determine the output string length
	linelength := 0
	for _, col := range columnWidths {
		linelength += col
	}

	// This is the actual line length.  It is the columnWidths (calculated in the
	// for loop above), extra spacing in the middle columns (ie, 2 * (# of cols - 2)
	// and one extra character for the newline.
	linelength = linelength + (spacing * (len(columnWidths) - 2) + 1)

	// If the line length is greater than the width of the terminal, truncate
	// the Title column.  The title text itself is truncated in the switch statement below.
	delta := linelength - width
	if delta > 0 {
		columnWidths[2] = columnWidths[2] - delta
	}

	// output the data to the screen
	for rownum, row := range data {
		weburl := row[0]
		for cellnum, cell := range row {
			if cellnum == 0 { // the 0th entry is the webURL and is not output
				continue
			}
			if rownum == 0 { // print out the header
				if cellnum < (len(row) - 1) {
					fmt.Printf("%-*s", columnWidths[cellnum]+spacing, cell)
				} else { // no spaces after last column
					fmt.Printf("%-*s", columnWidths[cellnum], cell)
				}
				continue
			}

			switch cellnum {
			case 1: // MRID (and weburl link)
				// Requires initial offset of width+spacing-len(cell)
				link := termlink.Link(cell, weburl)
				fmt.Printf("%s%-"+fmt.Sprintf("%d", columnWidths[cellnum]+spacing-len(cell))+"s",link, "")

			case 2: // MR Title
				fmt.Printf("%-*s", columnWidths[cellnum]+spacing, truncateText(cell, columnWidths[cellnum]))
			case 3: // CI Status
				switch cell {
				case "failed":
					printRED(cell, columnWidths[cellnum]+spacing)
				case "cancelled":
					printRED(cell, columnWidths[cellnum]+spacing)
				case "success":
					printGREEN(cell, columnWidths[cellnum]+spacing)
				case "running":
					printYELLOW(cell, columnWidths[cellnum]+spacing)
				default:
					printGREEN(cell, columnWidths[cellnum]+spacing)
				}

			case 4: // MR Status
				// spacing is not added here as this is the last column
				switch cell {
				case "mergeable":
					printGREEN(cell, columnWidths[cellnum])
				default:
					printRED(cell, columnWidths[cellnum])
				}
			}
		}
		fmt.Println()
	}
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list [remote] [search]",
	Aliases: []string{"ls", "search"},
	Short:   "List merge requests",
	Args:    cobra.MaximumNArgs(2),
	Example: heredoc.Doc(`
		lab mr list
		lab mr list "search terms"
		lab mr list --target-branch main
		lab mr list remote --target-branch main --label my-label
		lab mr list -l bug
		lab mr list -l close'
		lab mr list upstream -n 5
		lab mr list origin -a
		lab mr list --author johndoe
		lab mr list --assignee janedoe
		lab mr list --order created_at
		lab mr list --sort asc
		lab mr list --draft
		lab mr list --ready
		lab mr list --no-conflicts
		lab mr list -x 'test MR'
		lab mr list -r johndoe
		lab mr list --show-status`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, err := git.PathWithNamespace(defaultRemote)
		if err != nil {
			return
		}

		mrs, err := mrList(args)
		if err != nil {
			log.Fatal(err)
		}

		pager := newPager(cmd.Flags())
		defer pager.Close()

		showstatus, _ := cmd.Flags().GetBool("show-status")

		if !showstatus {
			for _, mr := range mrs {
				fmt.Printf("!%d %s\n", mr.IID, mr.Title)
			}
			return
		}

		output := [][]string{{"", "MRID", "Title", "CIStatus", "MRStatus"}}
		for _, mr := range mrs {
			mrx, err := lab.MRGet(rn, int(mr.IID))
			if err != nil {
				log.Fatal(err)
			}

			// In general we use the Detailed Merge Status.  There are some
			// cases below where a custom status is used.
			detailedMergeStatus := strings.Replace(mr.DetailedMergeStatus, "_", " ", -1)

			CIStatus := "no pipeline"
			if mrx.HeadPipeline != nil {
				CIStatus = mrx.HeadPipeline.Status
			}

			// Custom MR Status: If the status is success/not approved, then
			// check to see if there are any threads that need to be resolved.
			// If there are report 'unresolved threads' as a status
			if detailedMergeStatus == "not approved" && CIStatus == "success" {
				discussions, err := lab.MRListDiscussions(rn, mr.IID)
				if err != nil {
					log.Fatal(err)
				}

				totalresolved := 0
				totalresolvable := 0
				for _, discussion := range discussions {
					resolved := 0
					resolvable := 0
					for _, note := range discussion.Notes {
						if note.Resolved {
							resolved++
						}
						if note.Resolvable {
							resolvable++
						}
					}
					if resolved != 0 {
						totalresolved++
					}
					if resolvable != 0 {
						totalresolvable++
					}
				}
				if totalresolvable != 0 && totalresolvable != totalresolved {
					detailedMergeStatus = fmt.Sprintf("unresolved threads(%d/%d)", totalresolved, totalresolvable)
				}
			}

			// Custom Status: If the MR Status is 'not approved' also output
			// the number of remaining approvals necessary.
			if detailedMergeStatus == "not approved" {
				approvals, err := lab.GetMRApprovalsConfiguration(rn, mr.IID)
				if err != nil {
					log.Fatal(err)
				}
				detailedMergeStatus = fmt.Sprintf("%s(%d/%d)", detailedMergeStatus, len(approvals.ApprovedBy), approvals.ApprovalsRequired)
			}
			output = append(output,
					[]string{mr.WebURL, // weburl (used to convert MRID to URL)
						 strconv.Itoa(mr.IID), // MRID
						 mr.Title, // Title
						 CIStatus, // CI Status
						 detailedMergeStatus}) // MR Status
		}
		printColumns(output)
	},
}

func mrList(args []string) ([]*gitlab.BasicMergeRequest, error) {
	rn, search, err := parseArgsRemoteAndProject(args)
	if err != nil {
		return nil, err
	}

	labels, err := mapLabelsAsLabelOptions(rn, mrLabels)
	if err != nil {
		return nil, err
	}

	num, err := strconv.Atoi(mrNumRet)
	if mrAll || (err != nil) {
		num = -1
	}

	if mrApprover == "any" {
		mrApproverID = gitlab.ApproverIDs(gitlab.UserIDAny)
	} else if mrApprover == "none" {
		mrApproverID = gitlab.ApproverIDs(gitlab.UserIDNone)
	} else if mrApprover != "" {
		approverID := getUserID(mrApprover)
		if approverID == nil {
			log.Fatalf("%s user not found\n", mrApprover)
		}
		mrApproverID = gitlab.ApproverIDs([]int{*approverID})
	}

	// gitlab lib still doesn't have search by assignee and author username
	// for merge requests, because of that we need to get the ID for both.
	if mrAssignee == "any" {
		mrAssigneeID = gitlab.AssigneeID(gitlab.UserIDAny)
	} else if mrAssignee == "none" {
		mrAssigneeID = gitlab.AssigneeID(gitlab.UserIDNone)
	} else if mrAssignee != "" {
		assigneeID := getUserID(mrAssignee)
		if assigneeID == nil {
			log.Fatalf("%s user not found\n", mrAssignee)
		}
		mrAssigneeID = gitlab.AssigneeID(*assigneeID)
	} else if mrMine {
		assigneeID, err := lab.UserID()
		if err != nil {
			log.Fatal(err)
		}
		mrAssigneeID = gitlab.AssigneeID(assigneeID)
	}

	if mrAuthor != "" {
		mrAuthorID = getUserID(mrAuthor)
		if mrAuthorID == nil {
			log.Fatalf("%s user not found\n", mrAuthor)
		}
	}

	if strings.ToLower(mrMilestone) == "any" {
		mrMilestone = "Any"
	} else if strings.ToLower(mrMilestone) == "none" {
		mrMilestone = "None"
	} else if mrMilestone != "" {
		milestone, err := lab.MilestoneGet(rn, mrMilestone)
		if err != nil {
			log.Fatal(err)
		}
		mrMilestone = milestone.Title
	}

	if mrReviewer == "any" {
		mrReviewerID = gitlab.ReviewerID(gitlab.UserIDAny)
	} else if mrReviewer == "none" {
		mrReviewerID = gitlab.ReviewerID(gitlab.UserIDNone)
	} else if mrReviewer != "" {
		reviewerID := getUserID(mrReviewer)
		if reviewerID == nil {
			log.Fatalf("%s user not found\n", mrReviewer)
		}
		mrReviewerID = gitlab.ReviewerID(*reviewerID)
	}

	orderBy := gitlab.String(mrOrder)

	sort := gitlab.String(mrSortedBy)

	// if none of the flags are set, return every single MR
	mrCheckConflicts := (mrConflicts || mrNoConflicts)

	opts := gitlab.ListProjectMergeRequestsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: num,
		},
		Labels:                 &labels,
		State:                  &mrState,
		TargetBranch:           &mrTargetBranch,
		Milestone:              &mrMilestone,
		OrderBy:                orderBy,
		Sort:                   sort,
		AuthorID:               mrAuthorID,
		ApprovedByIDs:          mrApproverID,
		AssigneeID:             mrAssigneeID,
		WithMergeStatusRecheck: gitlab.Bool(mrCheckConflicts),
		ReviewerID:             mrReviewerID,
	}

	if mrDraft && !mrReady {
		opts.WIP = gitlab.String("yes")
	} else if mrReady && !mrDraft {
		opts.WIP = gitlab.String("no")
	}

	if mrExactMatch {
		if search == "" {
			return nil, errors.New("Exact match requested, but no search terms provided")
		}
		search = "\"" + search + "\""
	}

	if search != "" {
		opts.Search = &search
	}

	mrs, err := lab.MRList(rn, opts, num)
	if err != nil {
		return mrs, err
	}

	// only return MRs that matches the Conflicts requirement
	if mrCheckConflicts {
		var newMrList []*gitlab.BasicMergeRequest
		for _, mr := range mrs {
			if mr.HasConflicts && mrConflicts {
				newMrList = append(newMrList, mr)
			} else if !mr.HasConflicts && mrNoConflicts {
				newMrList = append(newMrList, mr)
			}
		}
		mrs = newMrList
	}

	return mrs, nil
}

func init() {
	listCmd.Flags().StringSliceVarP(
		&mrLabels, "label", "l", []string{}, "filter merge requests by label")
	listCmd.Flags().StringVarP(
		&mrState, "state", "s", "opened",
		"filter merge requests by state (all/opened/closed/merged)")
	listCmd.Flags().StringVarP(
		&mrNumRet, "number", "n", "10",
		"number of merge requests to return")
	listCmd.Flags().StringVarP(
		&mrTargetBranch, "target-branch", "t", "",
		"filter merge requests by target branch")
	listCmd.Flags().StringVar(
		&mrMilestone, "milestone", "", "list only MRs for the given milestone/any/none")
	listCmd.Flags().BoolVarP(&mrAll, "all", "a", false, "list all MRs on the project")
	listCmd.Flags().BoolVarP(&mrMine, "mine", "m", false, "list only MRs assigned to me")
	listCmd.Flags().MarkDeprecated("mine", "use --assignee instead")
	listCmd.Flags().StringVar(&mrAuthor, "author", "", "list only MRs authored by $username")
	listCmd.Flags().StringVar(
		&mrApprover, "approver", "", "list only MRs approved by $username/any/none")
	listCmd.Flags().StringVar(
		&mrAssignee, "assignee", "", "list only MRs assigned to $username/any/none")
	listCmd.Flags().StringVar(&mrOrder, "order", "updated_at", "display order (updated_at/created_at)")
	listCmd.Flags().StringVar(&mrSortedBy, "sort", "desc", "sort order (desc/asc)")
	listCmd.Flags().BoolVarP(&mrDraft, "draft", "", false, "list MRs marked as draft")
	listCmd.Flags().BoolVarP(&mrReady, "ready", "", false, "list MRs not marked as draft")
	listCmd.Flags().SortFlags = false
	listCmd.Flags().BoolVar(&mrNoConflicts, "no-conflicts", false, "list only MRs that can be merged")
	listCmd.Flags().BoolVar(&mrConflicts, "conflicts", false, "list only MRs that cannot be merged")
	listCmd.Flags().BoolVarP(&mrExactMatch, "exact-match", "x", false, "match on the exact (case-insensitive) search terms")
	listCmd.Flags().StringVar(
		&mrReviewer, "reviewer", "", "list only MRs with reviewer set to $username/any/none")
	listCmd.Flags().BoolP("show-status", "", false, "show CI and MR status (slow on projects with large number of MRs)")


	mrCmd.AddCommand(listCmd)
	carapace.Gen(listCmd).FlagCompletion(carapace.ActionMap{
		"label": carapace.ActionMultiParts(",", func(c carapace.Context) carapace.Action {
			project, _, err := parseArgsRemoteAndProject(c.Args)
			if err != nil {
				return carapace.ActionMessage(err.Error())
			}
			return action.Labels(project).Invoke(c).FilterParts()
		}),
		"milestone": carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			project, _, err := parseArgsRemoteAndProject(c.Args)
			if err != nil {
				return carapace.ActionMessage(err.Error())
			}
			return action.Milestones(project, action.MilestoneOpts{Active: true})
		}),
		"state": carapace.ActionValues("all", "opened", "closed", "merged"),
	})

	carapace.Gen(listCmd).PositionalCompletion(
		action.Remotes(),
	)
}
