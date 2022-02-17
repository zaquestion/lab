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

var (
	regTagsNonSemver bool
)

var regTagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List your registry tags",
	Example: heredoc.Doc(`
		lab reg tags 
		lab reg tags 99
		lab reg tags foo`),
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
					break
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

		nvs := make([]string, 0, len(registry.Tags))
		vs := make([]*semver.Version, 0, len(registry.Tags))
		for _, t := range registry.Tags {
			v, err := semver.NewVersion(t.Name)
			if err != nil {
				//log.Warnf("Error parsing version: %s %s", err, t.Name)
				nvs = append(nvs, t.Name)
				continue
			}
			vs = append(vs, v)
		}

		sort.Sort(semver.Collection(vs))

		if regTagsNonSemver {
			sort.Strings(nvs)
			for _, v := range nvs {
				fmt.Println(fmt.Sprintf("%s", v))
			}
		}
		for _, v := range vs {
			fmt.Println(fmt.Sprintf("%s", v))
		}
	},
}

func init() {
	regTagsCmd.Flags().BoolVarP(&regTagsNonSemver, "non-semver", "n", false, "also show non-semver version")
	regCmd.AddCommand(regTagsCmd)
}
