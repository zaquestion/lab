package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitconfig "github.com/tcnksm/go-gitconfig"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// mrCmd represents the mr command
var mrCreateCmd = &cobra.Command{
	Use:              "create [remote [branch]]",
	Aliases:          []string{"new"},
	Short:            "Open a merge request on GitLab",
	Long:             `Creates a merge request (default: MR created on default branch of origin)`,
	Args:             cobra.MaximumNArgs(2),
	PersistentPreRun: LabPersistentPreRun,
	Run:              runMRCreate,
}

func init() {
	mrCreateCmd.Flags().StringArrayP("message", "m", []string{}, "use the given <msg>; multiple -m are concatenated as separate paragraphs")
	mrCreateCmd.Flags().StringSliceP("assignee", "a", []string{}, "set assignee by username; can be specified multiple times for multiple assignees")
	mrCreateCmd.Flags().StringSliceP("label", "l", []string{}, "add label <label>; can be specified multiple times for multiple labels")
	mrCreateCmd.Flags().BoolP("remove-source-branch", "d", false, "remove source branch from remote after merge")
	mrCreateCmd.Flags().BoolP("squash", "s", false, "squash commits when merging")
	mrCreateCmd.Flags().Bool("allow-collaboration", false, "allow commits from other members")
	mrCreateCmd.Flags().Int("milestone", -1, "set milestone by milestone ID")
	mrCreateCmd.Flags().StringP("file", "F", "", "use the given file as the Description")
	mrCreateCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks")
	mrCreateCmd.Flags().BoolP("cover-letter", "c", false, "do not comment changelog and diffstat")
	mergeRequestCmd.Flags().AddFlagSet(mrCreateCmd.Flags())

	mrCmd.AddCommand(mrCreateCmd)
	carapace.Gen(mrCreateCmd).PositionalCompletion(
		action.Remotes(),
		action.RemoteBranches(0),
	)
}

// getAssignee returns the assigneeID for use with other GitLab API calls.
// NOTE: It is also used by issue_create.go
func getAssigneeID(assignee string) *int {
	if assignee == "" {
		return nil
	}
	if assignee[0] == '@' {
		assignee = assignee[1:]
	}
	assigneeID, err := lab.UserIDFromUsername(assignee)
	if err != nil {
		return nil
	}
	if assigneeID == -1 {
		return nil
	}
	return gitlab.Int(assigneeID)
}

// getAssignees returns the assigneeIDs for use with other GitLab API calls.
func getAssigneeIDs(assignees []string) []int {
	var ids []int
	for _, a := range assignees {
		ids = append(ids, *getAssigneeID(a))
	}
	return ids
}

func runMRCreate(cmd *cobra.Command, args []string) {
	msgs, err := cmd.Flags().GetStringArray("message")
	if err != nil {
		log.Fatal(err)
	}
	assignees, err := cmd.Flags().GetStringSlice("assignee")
	if err != nil {
		log.Fatal(err)
	}
	filename, err := cmd.Flags().GetString("file")
	if err != nil {
		log.Fatal(err)
	}
	branch, err := git.CurrentBranch()
	if err != nil {
		log.Fatal(err)
	}

	sourceRemote := determineSourceRemote(branch)
	sourceProjectName, err := git.PathWithNameSpace(sourceRemote)
	if err != nil {
		log.Fatal(err)
	}

	p, err := lab.FindProject(sourceProjectName)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := lab.GetCommit(p.ID, branch); err != nil {
		err = errors.Wrapf(
			err,
			"aborting MR, source branch %s not present on remote %s. did you forget to push?",
			branch, sourceRemote)
		log.Fatal(err)
	}

	targetRemote := forkedFromRemote
	if len(args) > 0 {
		targetRemote = args[0]
		ok, err := git.IsRemote(targetRemote)
		if err != nil || !ok {
			log.Fatal(errors.Wrapf(err, "%s is not a valid remote", targetRemote))
		}
	}
	targetProjectName, err := git.PathWithNameSpace(targetRemote)
	if err != nil {
		log.Fatal(err)
	}
	targetProject, err := lab.FindProject(targetProjectName)
	if err != nil {
		log.Fatal(err)
	}

	targetBranch := targetProject.DefaultBranch
	if len(args) > 1 && targetBranch != args[1] {
		targetBranch = args[1]
		if _, err := lab.GetCommit(targetProject.ID, targetBranch); err != nil {
			err = errors.Wrapf(
				err,
				"aborting MR, %s:%s is not a valid target. Did you forget to push %s to %s?",
				targetRemote, branch, branch, targetRemote)
			log.Fatal(err)
		}
	}

	var title, body string

	if filename != "" {
		file, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}

		fileScan := bufio.NewScanner(file)
		fileScan.Split(bufio.ScanLines)

		// The first line in the file is the title.
		fileScan.Scan()
		title = fileScan.Text()

		for fileScan.Scan() {
			body = body + fileScan.Text() + "\n"
		}

		file.Close()

	} else if len(msgs) > 0 {
		title, body = msgs[0], strings.Join(msgs[1:], "\n\n")
	} else {
		coverLetterFormat, _ := cmd.Flags().GetBool("cover-letter")
		msg, err := mrText(targetBranch, branch, sourceRemote, forkedFromRemote, coverLetterFormat)
		if err != nil {
			log.Fatal(err)
		}

		title, body, err = git.Edit("MERGEREQ", msg)
		if err != nil {
			_, f, l, _ := runtime.Caller(0)
			log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
		}
	}

	linebreak, _ := cmd.Flags().GetBool("force-linebreak")
	if linebreak {
		body = textToMarkdown(body)
	}

	removeSourceBranch, _ := cmd.Flags().GetBool("remove-source-branch")
	squash, _ := cmd.Flags().GetBool("squash")
	allowCollaboration, _ := cmd.Flags().GetBool("allow-collaboration")

	labels, err := cmd.Flags().GetStringSlice("label")
	if err != nil {
		log.Fatal(err)
	}

	milestoneID, _ := cmd.Flags().GetInt("milestone")
	var milestone *int
	if milestoneID < 0 {
		milestone = nil
	} else {
		milestone = &milestoneID
	}

	if title == "" {
		log.Fatal("aborting MR due to empty MR msg")
	}

	mrURL, err := lab.MRCreate(sourceProjectName, &gitlab.CreateMergeRequestOptions{
		SourceBranch:       &branch,
		TargetBranch:       gitlab.String(targetBranch),
		TargetProjectID:    &targetProject.ID,
		Title:              &title,
		Description:        &body,
		AssigneeIDs:        getAssigneeIDs(assignees),
		RemoveSourceBranch: &removeSourceBranch,
		Squash:             &squash,
		AllowCollaboration: &allowCollaboration,
		Labels:             labels,
		MilestoneID:        milestone,
	})
	if err != nil {
		// FIXME: not exiting fatal here to allow code coverage to
		// generate during Test_mrCreate. In the meantime API failures
		// will exit 0
		fmt.Fprintln(os.Stderr, err)
	}
	fmt.Println(mrURL + "/diffs")
}

func determineSourceRemote(branch string) string {
	// There is a precendence of options that should be considered here:
	// branch.<name>.pushRemote > remote.pushDefault > branch.<name>.remote
	// This rule is placed in git-config(1) manpage
	r, err := gitconfig.Local("branch." + branch + ".pushRemote")
	if err == nil {
		return r
	}
	r, err = gitconfig.Local("remote.pushDefault")
	if err == nil {
		return r
	}
	r, err = gitconfig.Local("branch." + branch + ".remote")
	if err == nil {
		return r
	}

	return forkRemote
}

func mrText(base, head, sourceRemote, forkedFromRemote string, coverLetterFormat bool) (string, error) {
	var (
		commitMsg string
		err       error
	)
	remoteBase := fmt.Sprintf("%s/%s", forkedFromRemote, base)
	commitMsg = ""
	if git.NumberCommits(remoteBase, head) == 1 {
		commitMsg, err = git.LastCommitMessage()
		if err != nil {
			return "", err
		}
	}

	const tmpl = `{{if .InitMsg}}{{.InitMsg}}{{end}}

{{if .Tmpl}}{{.Tmpl}}{{end}}
{{.CommentChar}} Requesting a merge into {{.Base}} from {{.Head}}
{{.CommentChar}}
{{.CommentChar}} Write a message for this merge request. The first block
{{.CommentChar}} of text is the title and the rest is the description.{{if .CommitLogs}}
{{.CommentChar}}
{{.CommentChar}} Changes:
{{.CommentChar}}
{{.CommitLogs}}{{end}}`

	mrTmpl := lab.LoadGitLabTmpl(lab.TmplMR)

	commitLogs, err := git.Log(remoteBase, head)
	if err != nil {
		return "", err
	}
	commitLogs = strings.TrimSpace(commitLogs)
	commentChar := git.CommentChar()

	if !coverLetterFormat {
		startRegexp := regexp.MustCompilePOSIX("^")
		commitLogs = startRegexp.ReplaceAllString(commitLogs, fmt.Sprintf("%s ", commentChar))
	} else {
		commitLogs = "\n" + strings.TrimSpace(commitLogs)
	}

	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return "", err
	}

	msg := &struct {
		InitMsg     string
		Tmpl        string
		CommentChar string
		Base        string
		Head        string
		CommitLogs  string
	}{
		InitMsg:     commitMsg,
		Tmpl:        mrTmpl,
		CommentChar: commentChar,
		Base:        forkedFromRemote + ":" + base,
		Head:        sourceRemote + ":" + head,
		CommitLogs:  commitLogs,
	}

	var b bytes.Buffer
	err = t.Execute(&b, msg)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}
