package cmd

import (
	"bufio"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	"github.com/zaquestion/lab/internal/config"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var issueShowCmd = &cobra.Command{
	Use:              "show [remote] <id>",
	Aliases:          []string{"get"},
	ArgAliases:       []string{"s"},
	Short:            "Describe an issue",
	Long:             ``,
	PersistentPreRun: LabPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {

		rn, issueNum, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		issue, err := lab.IssueGet(rn, int(issueNum))
		if err != nil {
			log.Fatal(err)
		}

		noMarkdown, _ := cmd.Flags().GetBool("no-markdown")
		if err != nil {
			log.Fatal(err)
		}
		renderMarkdown := !noMarkdown

		printIssue(issue, rn, renderMarkdown)

		showComments, _ := cmd.Flags().GetBool("comments")
		if showComments {
			discussions, err := lab.IssueListDiscussions(rn, int(issueNum))
			if err != nil {
				log.Fatal(err)
			}

			since, err := cmd.Flags().GetString("since")
			if err != nil {
				log.Fatal(err)
			}

			PrintDiscussions(discussions, since, "issues", int(issueNum))
		}
	},
}

func printIssue(issue *gitlab.Issue, project string, renderMarkdown bool) {
	milestone := "None"
	timestats := "None"
	dueDate := "None"
	state := map[string]string{
		"opened": "Open",
		"closed": "Closed",
	}[issue.State]
	if issue.Milestone != nil {
		milestone = issue.Milestone.Title
	}
	if issue.TimeStats != nil && issue.TimeStats.HumanTimeEstimate != "" &&
		issue.TimeStats.HumanTotalTimeSpent != "" {
		timestats = fmt.Sprintf(
			"Estimated %s, Spent %s",
			issue.TimeStats.HumanTimeEstimate,
			issue.TimeStats.HumanTotalTimeSpent)
	}
	if issue.DueDate != nil {
		dueDate = time.Time(*issue.DueDate).String()
	}
	assignees := make([]string, len(issue.Assignees))
	if len(issue.Assignees) > 0 && issue.Assignees[0].Username != "" {
		for i, a := range issue.Assignees {
			assignees[i] = a.Username
		}
	}

	if renderMarkdown {
		r, _ := glamour.NewTermRenderer(
			glamour.WithStandardStyle("auto"),
		)

		issue.Description, _ = r.Render(issue.Description)
	}

	fmt.Printf(`
#%d %s
===================================
%s
-----------------------------------
Project: %s
Status: %s
Assignees: %s
Author: %s
Milestone: %s
Due Date: %s
Time Stats: %s
Labels: %s
WebURL: %s
`,
		issue.IID, issue.Title, issue.Description, project, state, strings.Join(assignees, ", "),
		issue.Author.Username, milestone, dueDate, timestats,
		strings.Join(issue.Labels, ", "), issue.WebURL,
	)
}

// maxPadding returns the max value of two string numbers
func maxPadding(xstr string, ystr string) int {
	x, _ := strconv.Atoi(xstr)
	y, _ := strconv.Atoi(ystr)
	if x > y {
		return len(xstr)
	}
	return len(ystr)
}

// printDiffLine does a color print of a diff lines.  Red lines are removals
// and green lines are additions.
func printDiffLine(strColor string, maxChars int, sOld string, sNew string, ltext string) {

	switch strColor {
	case "":
		fmt.Printf("%*s %*s %s\n", maxChars, sOld, maxChars, sNew, ltext)
	case "green":
		color.Green("%*s %*s %s\n", maxChars, sOld, maxChars, sNew, ltext)
	case "red":
		color.Red("%*s %*s %s\n", maxChars, sOld, maxChars, sNew, ltext)
	}
}

// displayDiff displays the diff referenced in a discussion
func displayDiff(diff string, chunkNum int, newLine int, oldLine int) {
	var (
		diffChunkNum int = -1
		oldLineNum   int = 0
		newLineNum   int = 0
		maxChars     int
	)

	scanner := bufio.NewScanner(strings.NewReader(diff))
	for scanner.Scan() {
		if regexp.MustCompile(`^@@`).MatchString(scanner.Text()) {
			diffChunkNum++
			s := strings.Split(scanner.Text(), " ")
			dOld := strings.Split(s[1], ",")
			dNew := strings.Split(s[2], ",")

			// The patch can have, for example, either
			// @@ -1 +1 @@
			// or
			// @@ -1272,6 +1272,8 @@
			// so (len - 1) makes sense in both cases.
			maxChars = maxPadding(dOld[len(dOld)-1], dNew[len(dNew)-1]) + 1
			if diffChunkNum == chunkNum {
				continue
			}
			if diffChunkNum > chunkNum {
				break
			}
		}

		if diffChunkNum == chunkNum {
			var (
				sOld string
				sNew string
			)
			strColor := ""
			ltext := scanner.Text()
			ltag := string(ltext[0])
			switch ltag {
			case " ":
				strColor = ""
				oldLineNum++
				newLineNum++
				sOld = strconv.Itoa(oldLineNum)
				sNew = strconv.Itoa(newLineNum)
			case "-":
				strColor = "red"
				oldLineNum++
				sOld = strconv.Itoa(oldLineNum)
				sNew = " "
			case "+":
				strColor = "green"
				newLineNum++
				sOld = " "
				sNew = strconv.Itoa(newLineNum)
			}

			// output line
			if mrShowNoColorDiff {
				strColor = ""
			}
			if newLine != 0 {
				if newLineNum <= newLine && newLineNum >= (newLine-4) {
					printDiffLine(strColor, maxChars, sOld, sNew, ltext)
				}
			} else if oldLineNum <= oldLine && oldLineNum >= (oldLine-4) {
				printDiffLine(strColor, maxChars, sOld, sNew, ltext)
			}
		}
	}
}

func displayCommitDiscussion(idNum int, note *gitlab.Note) {

	// The GitLab API only supports showing comments on the entire
	// changeset and not per-commit.  IOW, all diffs are shown against
	// HEAD.  This is confusing in some scenarios, however it's what the
	// API provides.

	// Get a unified diff for the entire file
	diff, err := git.GetUnifiedDiff(note.Position.BaseSHA, note.Position.HeadSHA, note.Position.OldPath, note.Position.NewPath)
	if err != nil {
		fmt.Printf("    Could not get unified diff: Execute 'lab mr checkout %d; git checkout master' and try again.\n", idNum)
		return
	}

	if diff == "" {
		fmt.Println("    Could not find 'git diff' command.")
		return
	}

	// In general, only have to display the NewPath, however there
	// are some unusual cases where the OldPath may be displayed
	if note.Position.NewPath == note.Position.OldPath {
		fmt.Println("File:" + note.Position.NewPath)
	} else {
		fmt.Println("Files[old:" + note.Position.OldPath + " new:" + note.Position.NewPath + "]")
	}

	displayDiff(diff, 0, note.Position.NewLine, note.Position.OldLine)
	fmt.Println("")
	fmt.Println("")
}

func PrintDiscussions(discussions []*gitlab.Discussion, since string, idstr string, idNum int) {
	NewAccessTime := time.Now().UTC()

	issueEntry := fmt.Sprintf("%s%d", idstr, idNum)
	// if specified on command line use that, o/w use config, o/w Now
	var (
		CompareTime time.Time
		err         error
		sinceIsSet  = true
	)
	CompareTime, err = dateparse.ParseLocal(since)
	if err != nil || CompareTime.IsZero() {
		CompareTime = getMainConfig().GetTime(CommandPrefix + issueEntry)
		if CompareTime.IsZero() {
			CompareTime = time.Now().UTC()
		}
		sinceIsSet = false
	}

	// for available fields, see
	// https://godoc.org/github.com/xanzy/go-gitlab#Note
	// https://godoc.org/github.com/xanzy/go-gitlab#Discussion
	for _, discussion := range discussions {
		for i, note := range discussion.Notes {

			// skip system notes
			if note.System {
				continue
			}

			indentHeader, indentNote := "", ""
			commented := "commented"
			if !time.Time(*note.CreatedAt).Equal(time.Time(*note.UpdatedAt)) {
				commented = "updated comment"
			}

			if !discussion.IndividualNote {
				indentNote = "    "

				if i == 0 {
					commented = "started a discussion"
					if note.Position != nil {
						commented = "started a commit discussion"
					}
				} else {
					indentHeader = "    "
				}
			}

			noteBody := strings.Replace(note.Body, "\n", "\n"+indentHeader, -1)
			printit := color.New().PrintfFunc()
			printit(`
%s-----------------------------------`, indentHeader)

			if time.Time(*note.UpdatedAt).After(CompareTime) {
				printit = color.New(color.Bold).PrintfFunc()
			}
			printit(`
%s#%d: %s %s at %s

`,
				indentHeader, note.ID, note.Author.Username, commented, time.Time(*note.UpdatedAt).String())
			if note.Position != nil && i == 0 {
				displayCommitDiscussion(idNum, note)
			}
			printit(`%s%s
`,

				indentNote, noteBody)
		}
	}

	if sinceIsSet == false {
		config.WriteConfigEntry(CommandPrefix+issueEntry, NewAccessTime, "", "")
	}
}

func init() {
	issueShowCmd.Flags().BoolP("no-markdown", "M", false, "Don't use markdown renderer to print the issue description")
	issueShowCmd.Flags().BoolP("comments", "c", false, "Show comments for the issue")
	issueShowCmd.Flags().StringP("since", "s", "", "Show comments since specified date (format: 2020-08-21 14:57:46.808 +0000 UTC)")
	issueCmd.AddCommand(issueShowCmd)

	carapace.Gen(issueShowCmd).PositionalCompletion(
		action.Remotes(),
		action.Issues(issueList),
	)
}
