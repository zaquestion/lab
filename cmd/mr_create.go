package cmd

import (
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
	Use:     "create [remote [branch]]",
	Aliases: []string{"new"},
	Short:   "Open a merge request on GitLab",
	Long:    `Creates a merge request (MR created on origin master by default)`,
	Args:    cobra.MaximumNArgs(2),
	Run:     runMRCreate,
}

func init() {
	mrCreateCmd.Flags().StringSliceP("message", "m", []string{}, "Use the given <msg>; multiple -m are concatenated as separate paragraphs")
	mrCreateCmd.Flags().StringSliceP("assignee", "a", []string{}, "Set assignee by username; can be specified multiple times for multiple assignees")
	mrCreateCmd.Flags().StringSliceP("label", "l", []string{}, "Add label <label>; can be specified multiple times for multiple labels")
	mrCreateCmd.Flags().BoolP("remove-source-branch", "d", false, "Remove source branch from remote after merge")
	mrCreateCmd.Flags().BoolP("squash", "s", false, "Squash commits when merging")
	mrCreateCmd.Flags().Bool("allow-collaboration", false, "Allow commits from other members")
	mrCreateCmd.Flags().Int("milestone", -1, "Set milestone by milestone ID")
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
	msgs, err := cmd.Flags().GetStringSlice("message")
	if err != nil {
		log.Fatal(err)
	}
	assignees, err := cmd.Flags().GetStringSlice("assignee")
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
	targetBranch := "master"
	if len(args) > 1 && targetBranch != args[1] {
		targetBranch = args[1]
		if _, err := lab.GetCommit(targetProject.ID, targetBranch); err != nil {
			err = errors.Wrapf(
				err,
				"aborting MR, target branch %s not present on remote %s. did you forget to push?",
				targetBranch, targetRemote)
			log.Fatal(err)
		}
	}

	var title, body string

	if len(msgs) > 0 {
		title, body = msgs[0], strings.Join(msgs[1:], "\n\n")
	} else {
		msg, err := mrText(targetBranch, branch, sourceRemote, forkedFromRemote)
		if err != nil {
			log.Fatal(err)
		}

		title, body, err = git.Edit("MERGEREQ", msg)
		if err != nil {
			_, f, l, _ := runtime.Caller(0)
			log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
		}
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
		Labels:             lab.Labels(labels),
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
	// Check if the branch is being tracked
	r, err := gitconfig.Local("branch." + branch + ".remote")
	if err == nil {
		return r
	}

	return forkRemote
}

func mrText(base, head, sourceRemote, forkedFromRemote string) (string, error) {
	lastCommitMsg, err := git.LastCommitMessage()
	if err != nil {
		return "", err
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

	remoteBase := fmt.Sprintf("%s/%s", forkedFromRemote, base)
	commitLogs, err := git.Log(remoteBase, head)
	if err != nil {
		return "", err
	}
	startRegexp := regexp.MustCompilePOSIX("^")
	commentChar := git.CommentChar()
	commitLogs = strings.TrimSpace(commitLogs)
	commitLogs = startRegexp.ReplaceAllString(commitLogs, fmt.Sprintf("%s ", commentChar))

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
		InitMsg:     lastCommitMsg,
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
