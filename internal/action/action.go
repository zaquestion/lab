package action

import (
	"strconv"
	"strings"
	"time"

	"github.com/rsteube/carapace"
	"github.com/rsteube/carapace/pkg/cache"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// Remotes returns a carapace.Action containing all possible remote values
func Remotes() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		remotes, err := git.Remotes()
		if err != nil {
			return carapace.ActionMessage(err.Error())
		}
		return carapace.ActionValues(remotes...)
	})
}

// RemoteBranches returns a carapace.Action containing all possible remote
// branches values
func RemoteBranches(argIndex int) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		remote := ""
		if argIndex >= 0 {
			remote = c.Args[argIndex]
		}
		branches, err := git.RemoteBranches(remote)
		if err != nil {
			return carapace.ActionMessage(err.Error())
		}
		return carapace.ActionValues(branches...)
	})
}

// Snippets retuns a carapace.Action containing all available snippets
func Snippets(snippetList func(args []string) ([]*gitlab.Snippet, error)) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		snips, err := snippetList(c.Args[:0])
		if err != nil {
			return carapace.ActionMessage(err.Error())
		}

		values := make([]string, len(snips)*2)
		for index, snip := range snips {
			values[index*2] = strconv.Itoa(snip.ID)
			values[index*2+1] = snip.Title
		}
		return carapace.ActionValuesDescribed(values...)
	})
}

// Issues retuns a carapace.Action containing all available issues
func Issues(issueList func(args []string) ([]*gitlab.Issue, error)) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		issues, err := issueList(c.Args[:0])
		if err != nil {
			return carapace.ActionMessage(err.Error())
		}

		values := make([]string, len(issues)*2)
		for index, issue := range issues {
			values[index*2] = strconv.Itoa(issue.IID)
			values[index*2+1] = issue.Title
		}
		return carapace.ActionValuesDescribed(values...)
	})
}

// MergeRequests retuns a carapace.Action containing all available merge
// requests
func MergeRequests(mrList func(args []string) ([]*gitlab.MergeRequest, error)) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		mergeRequests, err := mrList(c.Args[:0])
		if err != nil {
			return carapace.ActionMessage(err.Error())
		}

		values := make([]string, len(mergeRequests)*2)
		for index, mergeRequest := range mergeRequests {
			values[index*2] = strconv.Itoa(mergeRequest.IID)
			values[index*2+1] = mergeRequest.Title
		}
		return carapace.ActionValuesDescribed(values...)
	})
}

// Labels returns a carapace.Action containing all possible labels
func Labels(project string) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		labels, err := lab.LabelList(project)
		if err != nil {
			return carapace.ActionMessage(err.Error())
		}

		values := make([]string, len(labels)*2)
		for index, label := range labels {
			values[index*2] = label.Name
			values[index*2+1] = label.Description
		}
		return carapace.ActionValuesDescribed(values...)
	}).Cache(5*time.Minute, cache.String(project))
}

// MilestoneOpts store filtering information for the milestones to be
// completed by Milestones().
type MilestoneOpts struct {
	Active bool
}

func (o MilestoneOpts) format() string {
	if o.Active {
		return "active"
	}
	return "closed"
}

// Milestones returns a carapace.Action containing all possible milestones with
// their description
func Milestones(project string, opts MilestoneOpts) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		state := opts.format()
		milestones, err := lab.MilestoneList(project, &gitlab.ListMilestonesOptions{State: &state})
		if err != nil {
			return carapace.ActionMessage(err.Error())
		}

		values := make([]string, len(milestones)*2)
		for index, milestone := range milestones {
			values[index*2] = milestone.Title
			values[index*2+1] = strings.SplitN(milestone.Description, "\n", 2)[0]
		}
		return carapace.ActionValuesDescribed(values...)
	}).Cache(5*time.Minute, cache.String(project, opts.format()))
}
