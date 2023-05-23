package cmd

import (
	"bytes"
	"fmt"
	"path/filepath"
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

var issueCreateCmd = &cobra.Command{
	Use:     "create [remote]",
	Aliases: []string{"new"},
	Short:   "Open an issue on GitLab",
	Args:    cobra.MaximumNArgs(1),
	Example: heredoc.Doc(`
		lab issue create
		lab issue create origin -a johndoe -a janedoe
		lab issue create origin -l bug
		lab issue create upstream -m "new issue related to the --help arg"
		lab issue create upstream --milestone "July"
		lab issue create upstream --template "API-BUG"`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		msgs, err := cmd.Flags().GetStringArray("message")
		if err != nil {
			log.Fatal(err)
		}
		assignees, err := cmd.Flags().GetStringSlice("assignees")
		if err != nil {
			log.Fatal(err)
		}
		labelTerms, err := cmd.Flags().GetStringSlice("label")
		if err != nil {
			log.Fatal(err)
		}
		milestoneName, err := cmd.Flags().GetString("milestone")
		if err != nil {
			log.Fatal(err)
		}
		templateName, err := cmd.Flags().GetString("template")
		if err != nil {
			log.Fatal(err)
		}
		remote := defaultRemote
		if len(args) > 0 {
			ok, err := git.IsRemote(args[0])
			if err != nil {
				log.Fatal(err)
			}
			if ok {
				remote = args[0]
			}
		}
		rn, err := git.PathWithNamespace(remote)
		if err != nil {
			log.Fatal(err)
		}

		labels, err := mapLabelsAsLabels(rn, labelTerms)
		if err != nil {
			log.Fatal(err)
		}

		var milestoneID *int
		if milestoneName != "" {
			milestone, err := lab.MilestoneGet(rn, milestoneName)
			if err != nil {
				log.Fatal(err)
			}
			milestoneID = &milestone.ID
		}

		title, body, err := issueMsg(templateName, msgs)
		if err != nil {
			_, f, l, _ := runtime.Caller(0)
			log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
		}
		if title == "" {
			log.Fatal("aborting issue due to empty issue msg")
		}

		linebreak, _ := cmd.Flags().GetBool("force-linebreak")
		if linebreak {
			body = textToMarkdown(body)
		}

		assigneeIDs := make([]int, len(assignees))
		for i, a := range assignees {
			assigneeIDs[i] = *getUserID(a)
		}

		issueURL, err := lab.IssueCreate(rn, &gitlab.CreateIssueOptions{
			Title:       &title,
			Description: &body,
			Labels:      &labels,
			AssigneeIDs: &assigneeIDs,
			MilestoneID: milestoneID,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(issueURL)
	},
}

func issueMsg(templateName string, msgs []string) (string, string, error) {
	if len(msgs) > 0 {
		return msgs[0], strings.Join(msgs[1:], "\n\n"), nil
	}

	text, err := issueText(templateName)
	if err != nil {
		return "", "", err
	}
	return git.Edit("ISSUE", text)
}

func issueText(templateName string) (string, error) {
	tmpl := heredoc.Doc(`
		{{.InitMsg}}
		{{.CommentChar}} Write a message for this issue. The first block
		{{.CommentChar}} of text is the title and the rest is the description.`)

	templateFile := filepath.Join("issue_templates", templateName)
	templateFile += ".md"
	issueTmpl := lab.LoadGitLabTmpl(templateFile)

	initMsg := "\n"
	if issueTmpl != "" {
		initMsg = "\n\n" + issueTmpl
	}

	commentChar := git.CommentChar()

	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return "", err
	}

	msg := &struct {
		InitMsg     string
		CommentChar string
	}{
		InitMsg:     initMsg,
		CommentChar: commentChar,
	}

	var b bytes.Buffer
	err = t.Execute(&b, msg)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

func init() {
	issueCreateCmd.Flags().StringArrayP("message", "m", []string{}, "use the given <msg>; multiple -m are concatenated as separate paragraphs")
	issueCreateCmd.Flags().StringSliceP("label", "l", []string{}, "set the given label(s) on the created issue")
	issueCreateCmd.Flags().StringSliceP("assignees", "a", []string{}, "set assignees by username")
	issueCreateCmd.Flags().String("milestone", "", "set milestone by title")
	issueCreateCmd.Flags().StringP("template", "t", "default", "use the given issue template")
	issueCreateCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks")

	issueCmd.AddCommand(issueCreateCmd)

	carapace.Gen(issueCreateCmd).FlagCompletion(carapace.ActionMap{
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

	carapace.Gen(issueCreateCmd).PositionalCompletion(
		action.Remotes(),
	)
}
