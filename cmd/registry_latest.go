package cmd

import (
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/Masterminds/semver"
	"sort"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var regLatestCmd = &cobra.Command{
	Use:   "latest",
	Short: "Show latest tag",
	Example: heredoc.Doc(`
		lab reg latest 
		lab reg latest foo
		lab reg latest 99`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, registryName, err := parseArgsRemoteAndProject(args)
		if err != nil {
			log.Fatal(err)
		}

		num, err := strconv.Atoi(projectListConfig.Number)
		if projectListConfig.All || (err != nil) {
			num = -1
		}

		project, err := lab.FindProject(rn)
		if err != nil {
			log.Fatal(err)
			return
		}

		opt := gitlab.ListRegistryRepositoriesOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: num,
			},
			Tags:      gitlab.Bool(true),
			TagsCount: gitlab.Bool(true),
		}
		registries, err := lab.ContainerRegistryList(project.ID, &opt, 0)
		if err != nil {
			log.Fatal(err)
			return
		}

		var registry *gitlab.RegistryRepository
		if len(registries) > 1 {
			if registryName == "" {
				log.Errorf("Found more than one registry, please specify")
				regListCmd.Run(cmd, args)
				return
			}

			registryId, err := strconv.Atoi(registryName)

			for _, r := range registries {
				if r.Path == registryName || (err == nil && r.ID == registryId) {
					registry = r
				}
			}

			if registry == nil {
				log.Errorf("Registry %s not found", registryName)
				return

			}
		} else {
			registry = registries[0]
		}

		log.Debugf("Using registry %s (%d)\n", registry.Path, registry.ID)

		vs := make([]*semver.Version, 0, len(registry.Tags))
		for _, t := range registry.Tags {
			v, err := semver.NewVersion(t.Name)
			if err != nil {
				//log.Warnf("Error parsing version: %s %s", err, t.Name)
				continue
			}
			vs = append(vs, v)
		}
		//fmt.Printf("%+v\n", vs)
		sort.Sort(semver.Collection(vs))
		fmt.Println(vs[len(vs)-1].String())
	},
}

func init() {
	regCmd.AddCommand(regLatestCmd)
}
