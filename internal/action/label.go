package action

import (
	"time"

	"github.com/rsteube/carapace"
	"github.com/rsteube/carapace/pkg/cache"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

func Labels(project string) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		if labels, err := lab.LabelList(project); err != nil {
			return carapace.ActionMessage(err.Error())
		} else {
			values := make([]string, len(labels)*2)
			for index, label := range labels {
				values[index*2] = label.Name
				values[index*2+1] = label.Description
			}
			return carapace.ActionValuesDescribed(values...)
		}
	}).Cache(5*time.Minute, cache.String(project))
}
