package cmd

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
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

// mrCmd represents the mr command
var mrCreateCmd = &cobra.Command{
	Use:     "create [target_remote [target_branch]]",
	Aliases: []string{"new"},
	Short:   "Creates a merge request.",
	Args:    cobra.MaximumNArgs(2),
	Example: heredoc.Doc(`
		lab mr create target_remote
		lab mr create target_remote target_branch --allow-collaboration
		lab mr create upstream main --source my_fork:feature-3
		lab mr create a_remote -a johndoe -a janedoe
		lab mr create my_remote -c
		lab mr create my_remote --draft
		lab mr create my_remote -F a_file.txt
		lab mr create my_remote -F a_file.txt --force-linebreak
		lab mr create my_remote -f a_file.txt
		lab mr create my_remote -l bug -l confirmed
		lab mr create my_remote -m "A title message"
		lab mr create my_remote -m "A MR title" -m "A MR description"
		lab mr create my_remote --milestone "Fall"
		lab mr create my_remote -d
		lab mr create my_remote -r johndoe -r janedoe
		lab mr create my_remote -s`),
	PersistentPreRun: labPersistentPreRun,
	Run:              runMRCreate,
}

func init() {
	mrCreateCmd.Flags().StringArrayP("message", "m", []string{}, "use the given <msg>; multiple -m are concatenated as separate paragraphs")
	mrCreateCmd.Flags().StringSliceP("assignee", "a", []string{}, "set assignee by username; can be specified multiple times for multiple assignees")
	mrCreateCmd.Flags().StringSliceP("reviewer", "r", []string{}, "set reviewer by username; can be specified multiple times for multiple reviewers")
	mrCreateCmd.Flags().StringSliceP("label", "l", []string{}, "add label <label>; can be specified multiple times for multiple labels")
	mrCreateCmd.Flags().BoolP("remove-source-branch", "d", false, "remove source branch from remote after merge")
	mrCreateCmd.Flags().BoolP("squash", "s", false, "squash commits when merging")
	mrCreateCmd.Flags().Bool("allow-collaboration", false, "allow commits from other members")
	mrCreateCmd.Flags().String("milestone", "", "set milestone by milestone title or ID")
	mrCreateCmd.Flags().StringP("file", "F", "", "use the given file as the Title and Description")
	mrCreateCmd.Flags().StringP("file-edit", "f", "", "use the given file as the Title and Description and open the editor")
	mrCreateCmd.Flags().Bool("no-edit", false, "use the selected commit message without opening the editor")
	mrCreateCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks")
	mrCreateCmd.Flags().BoolP("cover-letter", "c", false, "comment changelog and diffstat")
	mrCreateCmd.Flags().Bool("draft", false, "mark the merge request as draft")
	mrCreateCmd.Flags().String("source", "", "specify the source remote and branch in the form of remote:branch")
	mergeRequestCmd.Flags().AddFlagSet(mrCreateCmd.Flags())

	mrCmd.AddCommand(mrCreateCmd)

	carapace.Gen(mrCreateCmd).FlagCompletion(carapace.ActionMap{
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

	carapace.Gen(mrCreateCmd).PositionalCompletion(
		action.Remotes(),
		action.RemoteBranches(0),
	)
}

func verifyRemoteBranch(projID string, branch string) error {
	if _, err := lab.GetCommit(projID, branch); err != nil {
		return fmt.Errorf("%s is not a valid reference", branch)
	}
	return nil
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
	reviewers, err := cmd.Flags().GetStringSlice("reviewer")
	if err != nil {
		log.Fatal(err)
	}

	filename, err := cmd.Flags().GetString("file")
	if err != nil {
		log.Fatal(err)
	}

	ofilename, err := cmd.Flags().GetString("file-edit")
	if err != nil {
		log.Fatal(err)
	}

	if ofilename != "" && filename != "" {
		log.Fatalf("Cannot specify both -F and -f options.")
	}

	noEdit, err := cmd.Flags().GetBool("no-edit")
	if err != nil {
		log.Fatal(err)
	}

	coverLetterFormat, err := cmd.Flags().GetBool("cover-letter")
	if err != nil {
		log.Fatal(err)
	}

	localBranch, err := git.CurrentBranch()
	if err != nil {
		log.Fatal(err)
	}

	sourceRemote, err := determineSourceRemote(localBranch)
	if err != nil {
		log.Fatal(err)
	}

	// Get the pushed branch name
	sourceBranch, _ := git.UpstreamBranch(localBranch)
	if sourceBranch == "" {
		// Fall back to local branch name
		sourceBranch = localBranch
	}

	sourceTarget, err := cmd.Flags().GetString("source")
	if err != nil {
		log.Fatal(err)
	}

	if sourceTarget != "" {
		sourceParts := strings.Split(sourceTarget, ":")
		if len(sourceParts) < 2 ||
			strings.TrimSpace(sourceParts[0]) == "" ||
			strings.TrimSpace(sourceParts[1]) == "" {
			log.Fatalf("source remote must have format remote:remote_branch")
		}

		sourceRemote = sourceParts[0]
		sourceBranch = sourceParts[1]

		_, err := git.IsRemote(sourceRemote)
		if err != nil {
			log.Fatal(err)
		}
	}

	sourceProjectName, err := git.PathWithNamespace(sourceRemote)
	if err != nil {
		log.Fatal(err)
	}

	// verify the source branch in remote project
	err = verifyRemoteBranch(sourceProjectName, sourceBranch)
	if err != nil {
		log.Fatalf("%s:%s\n", sourceRemote, err)
	}

	targetRemote := defaultRemote
	if len(args) > 0 {
		targetRemote = args[0]
		ok, err := git.IsRemote(targetRemote)
		if err != nil || !ok {
			log.Fatalf("%s is not a valid remote\n", targetRemote)
		}
	}
	targetProjectName, err := git.PathWithNamespace(targetRemote)
	if err != nil {
		log.Fatal(err)
	}
	targetProject, err := lab.FindProject(targetProjectName)
	if err != nil {
		if err == lab.ErrProjectNotFound {
			log.Fatalf("GitLab project (%s) not found, verify you have access to the requested resource", targetProjectName)
		}
		log.Fatal(err)
	}

	targetBranch := targetProject.DefaultBranch
	if len(args) > 1 && targetBranch != args[1] {
		targetBranch = args[1]
		// verify the target branch in remote project
		err = verifyRemoteBranch(targetProjectName, targetBranch)
		if err != nil {
			log.Fatalf("%s:%s\n", targetRemote, err)
		}
	}

	labelTerms, err := cmd.Flags().GetStringSlice("label")
	if err != nil {
		log.Fatal(err)
	}
	labels, err := mapLabels(targetProjectName, labelTerms)
	if err != nil {
		log.Fatal(err)
	}

	milestoneArg, _ := cmd.Flags().GetString("milestone")
	milestoneID, _ := strconv.Atoi(milestoneArg)

	var milestone *int
	if milestoneID > 0 {
		milestone = &milestoneID
	} else if milestoneArg != "" {
		ms, err := lab.MilestoneGet(targetProjectName, milestoneArg)
		if err != nil {
			log.Fatal(err)
		}
		milestone = &ms.ID
	} else {
		milestone = nil
	}

	var title, body string

	if filename != "" {
		var openEditor bool

		if ofilename != "" {
			filename = ofilename
			openEditor = true
		}

		if _, err := os.Stat(filename); os.IsNotExist(err) {
			log.Fatalf("file %s cannot be found", filename)
		}

		if len(msgs) > 0 || coverLetterFormat {
			log.Fatal("option -F cannot be combined with -m/-c")
		}

		title, body, err = editDescription("", "", nil, filename)
		if err != nil {
			log.Fatal(err)
		}

		if openEditor {
			msg, err := mrText(sourceRemote, sourceBranch, targetRemote,
				targetBranch, coverLetterFormat, false)
			if err != nil {
				log.Fatal(err)
			}

			msg = title + body + msg

			title, body, err = git.Edit("MERGEREQ", msg)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else if len(msgs) > 0 {
		if coverLetterFormat {
			log.Fatal("option -m cannot be combined with -c/-F")
		}

		title, body = msgs[0], strings.Join(msgs[1:], "\n\n")
	} else {
		msg, err := mrText(sourceRemote, sourceBranch, targetRemote,
			targetBranch, coverLetterFormat, true)
		if err != nil {
			log.Fatal(err)
		}

		openEditor := !noEdit
		if openEditor {
			title, body, err = git.Edit("MERGEREQ", msg)
			if err != nil {
				_, f, l, _ := runtime.Caller(0)
				log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
			}
		} else {
			title, body, err = git.ParseTitleBody(msg)
			if title == "" {
				// The only way to get here is if the auto-generated MR
				// description has no title or body, which happens when the
				// MR is made of multiple commits instead of a single one
				// where the commit log is used as the MR description.
				log.Fatal("use of --no-edit with multiple commits not allowed")
			}
		}
	}

	if title == "" {
		log.Fatal("empty MR message")
	}

	linebreak, _ := cmd.Flags().GetBool("force-linebreak")
	if linebreak {
		body = textToMarkdown(body)
	}

	draft, _ := cmd.Flags().GetBool("draft")
	if draft {
		// GitLab 14.0 will remove WIP support in favor of Draft
		isWIP := hasPrefix(title, "wip:") ||
			hasPrefix(title, "[wip]")
		if isWIP {
			log.Fatal("the use of \"WIP\" terminology is deprecated, use \"Draft\" instead")
		}

		isDraft := hasPrefix(title, "draft:") ||
			hasPrefix(title, "[draft]") ||
			hasPrefix(title, "(draft)")
		if !isDraft {
			title = "Draft: " + title
		}
	}

	removeSourceBranch, _ := cmd.Flags().GetBool("remove-source-branch")
	squash, _ := cmd.Flags().GetBool("squash")
	allowCollaboration, _ := cmd.Flags().GetBool("allow-collaboration")

	mrURL, err := lab.MRCreate(sourceProjectName, &gitlab.CreateMergeRequestOptions{
		SourceBranch:       &sourceBranch,
		TargetBranch:       gitlab.String(targetBranch),
		TargetProjectID:    &targetProject.ID,
		Title:              &title,
		Description:        &body,
		AssigneeIDs:        getUserIDs(assignees),
		ReviewerIDs:        getUserIDs(reviewers),
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
	} else {
		fmt.Println(mrURL + "/diffs")
	}
}

func mrText(sourceRemote, sourceBranch, targetRemote, targetBranch string, coverLetterFormat bool, generateCommitMsg bool) (string, error) {
	target := fmt.Sprintf("%s/%s", targetRemote, targetBranch)
	source := fmt.Sprintf("%s/%s", sourceRemote, sourceBranch)
	commitMsg := ""

	numCommits := git.NumberCommits(target, source)
	if numCommits == 1 && generateCommitMsg {
		var err error
		commitMsg, err = git.LastCommitMessage(source)
		if err != nil {
			return "", err
		}
	}
	if numCommits == 0 {
		return "", fmt.Errorf("the resulting MR from %s to %s has 0 commits", target, source)
	}

	tmpl := heredoc.Doc(`
		{{if .InitMsg}}{{.InitMsg}}{{end}}

		{{if .Tmpl}}{{.Tmpl}}{{end}}
		{{.CommentChar}} Requesting a merge into {{.Target}} from {{.Source}} ({{.NumCommits}} commits)
		{{.CommentChar}}
		{{.CommentChar}} Write a message for this merge request. The first block
		{{.CommentChar}} of text is the title and the rest is the description.{{if .CommitLogs}}
		{{.CommentChar}}
		{{.CommentChar}} Changes:
		{{.CommentChar}}
		{{.CommitLogs}}{{end}}
	`)

	mrTmpl := lab.LoadGitLabTmpl(lab.TmplMR)

	commitLogs, err := git.Log(target, source)
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
		Target      string
		Source      string
		CommitLogs  string
		NumCommits  int
	}{
		InitMsg:     commitMsg,
		Tmpl:        mrTmpl,
		CommentChar: commentChar,
		Target:      targetRemote + ":" + targetBranch,
		Source:      sourceRemote + ":" + sourceBranch,
		CommitLogs:  commitLogs,
		NumCommits:  numCommits,
	}

	var b bytes.Buffer
	err = t.Execute(&b, msg)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}
