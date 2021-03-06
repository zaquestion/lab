package action

import (
	"strings"
	"time"

	"github.com/rsteube/carapace"
	"github.com/rsteube/carapace/pkg/cache"
	"github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

type MilestoneOpts struct {
	Active bool
}

func (o MilestoneOpts) format() string {
	if o.Active {
		return "active"
	} else {
		return "closed"
	}
}

func Milestones(project string, opts MilestoneOpts) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		state := opts.format()
		if milestones, err := lab.MilestoneList(project, &gitlab.ListMilestonesOptions{State: &state}); err != nil {
			return carapace.ActionMessage(err.Error())
		} else {
			values := make([]string, len(milestones)*2)
			for index, milestone := range milestones {
				values[index*2] = milestone.Title
				values[index*2+1] = strings.SplitN(milestone.Description, "\n", 2)[0]
			}
			return carapace.ActionValuesDescribed(values...)
		}
	}).Cache(5*time.Minute, cache.String(project, opts.format()))
}
