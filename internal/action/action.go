package action

import (
	"strconv"

	"github.com/rsteube/carapace"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
)

func Remotes() carapace.Action {
	return carapace.ActionCallback(func(args []string) carapace.Action {
		if remotes, err := git.Remotes(); err != nil {
			return carapace.ActionMessage(err.Error())
		} else {
			return carapace.ActionValues(remotes...)
		}
	})
}

func RemoteBranches(argIndex int) carapace.Action {
	return carapace.ActionCallback(func(args []string) carapace.Action {
		remote := ""
		if argIndex >= 0 {
			remote = args[argIndex]
		}
		if branches, err := git.RemoteBranches(remote); err != nil {
			return carapace.ActionMessage(err.Error())
		} else {
			return carapace.ActionValues(branches...)
		}
	})
}

func Snippets(snippetList func(args []string) ([]*gitlab.Snippet, error)) carapace.Action {
	return carapace.ActionCallback(func(args []string) carapace.Action {
		if snips, err := snippetList(args[:0]); err != nil {
			return carapace.ActionMessage(err.Error())
		} else {
			values := make([]string, len(snips)*2)
			for index, snip := range snips {
				values[index*2] = strconv.Itoa(snip.ID)
				values[index*2+1] = snip.Title
			}
			return carapace.ActionValuesDescribed(values...)
		}
	})
}

func Issues(issueList func(args []string) ([]*gitlab.Issue, error)) carapace.Action {
	return carapace.ActionCallback(func(args []string) carapace.Action {
		if issues, err := issueList(args[:0]); err != nil {
			return carapace.ActionMessage(err.Error())
		} else {
			values := make([]string, len(issues)*2)
			for index, issue := range issues {
				values[index*2] = strconv.Itoa(issue.IID)
				values[index*2+1] = issue.Title
			}
			return carapace.ActionValuesDescribed(values...)
		}
	})
}

func MergeRequests(mrList func(args []string) ([]*gitlab.MergeRequest, error)) carapace.Action {
	return carapace.ActionCallback(func(args []string) carapace.Action {
		if mergeRequests, err := mrList(args[:0]); err != nil {
			return carapace.ActionMessage(err.Error())
		} else {
			values := make([]string, len(mergeRequests)*2)
			for index, mergeRequest := range mergeRequests {
				values[index*2] = strconv.Itoa(mergeRequest.IID)
				values[index*2+1] = mergeRequest.Title
			}
			return carapace.ActionValuesDescribed(values...)
		}
	})
}
