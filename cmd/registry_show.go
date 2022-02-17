package cmd

import (
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var regShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Describe registry",
	Example: heredoc.Doc(`
		lab reg show 
		lab reg show foo
		lab reg show 99`),
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

		project, _ := lab.FindProject(rn)
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

		//fmt.Printf("%+v\n", registry)
		fmt.Printf("ID:       %d\n", registry.ID)
		fmt.Printf("Name:     %s\n", registry.Name)
		fmt.Printf("Path:     %s\n", registry.Path)
		fmt.Printf("Location: %s\n", registry.Location)
		fmt.Printf("Created:  %s\n", registry.CreatedAt)
		fmt.Printf("# tags:   %d\n", registry.TagsCount)
		if registry.CleanupPolicyStartedAt != nil {
			fmt.Printf("CleanupPolicy started at: %s\n", registry.CleanupPolicyStartedAt)
		}
	},
}

func init() {
	regCmd.AddCommand(regShowCmd)
}
