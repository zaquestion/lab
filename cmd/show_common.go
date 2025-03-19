package cmd

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/araddon/dateparse"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/fatih/color"
	"github.com/jaytaylor/html2text"
	"github.com/muesli/termenv"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"github.com/zaquestion/lab/internal/config"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

func inRange(val int, min int, max int) bool {
	return val >= min && val <= max
}

// maxPadding returns the max value of two string numbers
func maxPadding(x int, y int) int {
	if x > y {
		return len(strconv.Itoa(x))
	}
	return len(strconv.Itoa(y))
}

// printDiffLine does a color print of a diff lines.  Red lines are removals
// and green lines are additions.
func printDiffLine(strColor string, maxChars int, sOld string, sNew string, ltext string) string {

	switch strColor {
	case "green":
		return color.GreenString("|%*s %*s %s\n", maxChars, sOld, maxChars, sNew, ltext)
	case "red":
		return color.RedString("|%*s %*s %s\n", maxChars, sOld, maxChars, sNew, ltext)
	}
	return fmt.Sprintf("|%*s %*s %s\n", maxChars, sOld, maxChars, sNew, ltext)
}

// displayDiff displays the diff referenced in a discussion
func displayDiff(diff string, newLine int, oldLine int, outputAll bool) string {
	var (
		oldLineNum int = 0
		newLineNum int = 0
		maxChars   int
		output     bool   = false
		diffOutput string = ""
	)

	scanner := bufio.NewScanner(strings.NewReader(diff))
	for scanner.Scan() {
		if regexp.MustCompile(`^@@`).MatchString(scanner.Text()) {
			s := strings.Split(scanner.Text(), " ")
			dOld := strings.Split(s[1], ",")
			dNew := strings.Split(s[2], ",")

			// get the new line number of the first line of the diff
			newDiffStart, err := strconv.Atoi(strings.Replace(dNew[0], "+", "", -1))
			if err != nil {
				log.Fatal(err)
			}
			newDiffRange := 1
			if len(dNew) == 2 {
				newDiffRange, err = strconv.Atoi(dNew[1])
				if err != nil {
					log.Fatal(err)
				}
			}
			newDiffEnd := newDiffStart + newDiffRange
			newLineNum = newDiffStart - 1

			// get the old line number of the first line of the diff
			oldDiffStart, err := strconv.Atoi(strings.Replace(dOld[0], "-", "", -1))
			if err != nil {
				log.Fatal(err)
			}

			oldDiffEnd := newDiffRange
			oldLineNum = oldDiffStart - 1
			if len(dOld) > 1 {
				oldDiffRange, err := strconv.Atoi(dOld[1])
				if err != nil {
					log.Fatal(err)
				}
				oldDiffEnd = oldDiffStart + oldDiffRange
			}

			if (oldLine != 0) && inRange(oldLine, oldDiffStart, oldDiffEnd) {
				output = true
			} else if (newLine != 0) && inRange(newLine, newDiffStart, newDiffEnd) {
				output = true
			} else {
				output = false
			}

			if outputAll {
				mrShowNoColorDiff = true
				output = true
			}

			// padding to align diff output (depends on the line numbers' length)
			// The patch can have, for example, either
			// @@ -1 +1 @@
			// or
			// @@ -1272,6 +1272,8 @@
			// so (len - 1) makes sense in both cases.
			maxChars = maxPadding(oldDiffEnd, newDiffEnd) + 1
		}

		if !output {
			continue
		}

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
		if outputAll {
			diffOutput += printDiffLine(strColor, maxChars, sOld, sNew, ltext)
		} else if newLine != 0 {
			if newLineNum <= newLine && newLineNum >= (newLine-4) {
				diffOutput += printDiffLine(strColor, maxChars, sOld, sNew, ltext)
			}
		} else if oldLineNum <= oldLine && oldLineNum >= (oldLine-4) {
			diffOutput += printDiffLine(strColor, maxChars, sOld, sNew, ltext)
		}
	}
	return diffOutput
}

func displayCommitDiscussion(project string, idNum int, note *gitlab.Note) {

	// Previously, the GitLab API only supports showing comments on the
	// entire changeset and not per-commit.  IOW, all diffs were shown
	// against HEAD.  This was confusing in some scenarios, however it
	// was what the API provided.
	//
	// At some point, note.Position.HeadSHA was changed to report the
	// commit that was being commented on, and note.Position.BaseSHA
	// points to the commit just before it.  I cannot find the GitLab
	// commit that changed this behaviour but it has changed.
	//
	// I am leaving this comment here in case it ever changes again.

	// In some cases the CommitID field is still not populated correctly.
	// In those cases use the HeadSHA value instead of the CommitID.
	commitID := note.CommitID
	if commitID == "" {
		commitID = note.Position.HeadSHA
	}

	// Get a unified diff for the entire file
	ds, err := lab.GetCommitDiff(project, commitID)
	if err != nil {
		fmt.Printf("    Could not get diff for commit %s.\n", commitID)
		return
	}

	if len(ds) == 0 {
		log.Fatal("    No diff found for %s.", commitID)
	}

	// In general, only have to display the NewPath, however there
	// are some unusual cases where the OldPath may be displayed
	if note.Position.NewPath == note.Position.OldPath {
		fmt.Println("commit:" + commitID)
		fmt.Println("File:" + note.Position.NewPath)
	} else {
		fmt.Println("commit:" + commitID)
		fmt.Println("Files[old:" + note.Position.OldPath + " new:" + note.Position.NewPath + "]")
	}

	for _, d := range ds {
		if note.Position.NewPath == d.NewPath && note.Position.OldPath == d.OldPath {
			newLine := note.Position.NewLine
			oldLine := note.Position.OldLine
			if note.Position.LineRange != nil {
				newLine = note.Position.LineRange.StartRange.NewLine
				oldLine = note.Position.LineRange.StartRange.OldLine
			}
			diffstring := displayDiff(d.Diff, newLine, oldLine, false)
			fmt.Printf(diffstring)
		}
	}
	fmt.Println("")
}

func getBoldStyle() ansi.StyleConfig {
	var style ansi.StyleConfig
	if termenv.HasDarkBackground() {
		style = glamour.DarkStyleConfig
	} else {
		style = glamour.LightStyleConfig
	}
	bold := true
	style.Document.Bold = &bold
	return style
}

func getTermRenderer(style glamour.TermRendererOption) (*glamour.TermRenderer, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithWordWrap(0),
		// There are PAGERs and TERMs that supports only 16 colors,
		// since we aren't a beauty-driven project, lets use it.
		glamour.WithColorProfile(termenv.ANSI),
		style,
	)
	return r, err
}

const (
	NoteLevelNone = iota
	NoteLevelComments
	NoteLevelActivities
	NoteLevelFull
)

func printDiscussions(project string, discussions []*gitlab.Discussion, since string, idstr string, idNum int, renderMarkdown bool, noteLevel int) {
	newAccessTime := time.Now().UTC()

	issueEntry := fmt.Sprintf("%s%d", idstr, idNum)
	// if specified on command line use that, o/w use config, o/w Now
	var (
		comparetime time.Time
		err         error
		sinceIsSet  = true
	)
	comparetime, err = dateparse.ParseLocal(since)
	if err != nil || comparetime.IsZero() {
		comparetime = getMainConfig().GetTime(commandPrefix + issueEntry)
		if comparetime.IsZero() {
			comparetime = time.Now().UTC()
		}
		sinceIsSet = false
	}

	mdRendererNormal, _ := getTermRenderer(glamour.WithAutoStyle())
	mdRendererBold, _ := getTermRenderer(glamour.WithStyles(getBoldStyle()))

	// for available fields, see
	// https://godoc.org/github.com/xanzy/go-gitlab#Note
	// https://godoc.org/github.com/xanzy/go-gitlab#Discussion
	for _, discussion := range discussions {
		for i, note := range discussion.Notes {
			if (noteLevel == NoteLevelActivities && note.System == false) ||
				(noteLevel == NoteLevelComments && note.System == true) {
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

			noteBody := strings.Replace(note.Body, "\n", "<br>\n", -1)
			html2textOptions := html2text.Options{
				PrettyTables: true,
				OmitLinks:    true,
			}
			noteBody, _ = html2text.FromString(noteBody, html2textOptions)
			mdRenderer := mdRendererNormal
			printit := color.New().PrintfFunc()
			if note.System {
				splitNote := strings.SplitN(noteBody, "\n", 2)

				// system notes are informational messages only
				// and cannot have replies.  Do not output the
				// note.ID
				printit(
					heredoc.Doc(`
						* %s %s at %s
					`),
					note.Author.Username, splitNote[0], time.Time(*note.UpdatedAt).String())
				if len(splitNote) == 2 {
					if renderMarkdown {
						splitNote[1], _ = mdRenderer.Render(splitNote[1])
					}
					printit(
						heredoc.Doc(`
							%s
						`),
						splitNote[1])
				}
				continue
			}

			printit(
				heredoc.Doc(`
					%s-----------------------------------
				`),
				indentHeader)

			if time.Time(*note.UpdatedAt).After(comparetime) {
				mdRenderer = mdRendererBold
				printit = color.New(color.Bold).PrintfFunc()
			}

			if renderMarkdown {
				noteBody, _ = mdRenderer.Render(noteBody)
			}
			noteBody = strings.Replace(noteBody, "\n", "\n"+indentNote, -1)

			printit(`%s#%d: %s %s at %s

`,
				indentHeader, note.ID, note.Author.Username, commented, time.Time(*note.UpdatedAt).String())
			if note.Position != nil && i == 0 {
				displayCommitDiscussion(project, idNum, note)
			}
			printit(`%s%s
`,

				indentNote, noteBody)
		}
	}

	if !sinceIsSet {
		config.WriteConfigEntry(commandPrefix+issueEntry, newAccessTime, "", "")
	}
}
