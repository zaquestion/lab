package cmd

import (
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var regTagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Describe registry tag",
	Example: heredoc.Doc(`
		lab reg tag 
		lab reg tag 1.2.3
		lab reg tag 99@1.2.3
		lab reg tags foo@1.2.3`),
	PersistentPreRun: labPersistentPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		rn, tagName, err := parseArgsRemoteAndProject(args)
		if err != nil {
			log.Fatal(err)
		}

		registryName := ""
		if strings.Contains(tagName, "@") {
			s := strings.Split(tagName, "@")
			registryName = s[0]
			tagName = s[1]
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

		tag, err := lab.ContainerRegistryTagDetail(project.ID, registry.ID, tagName)

		if err != nil {
			log.Errorf("Tag %s not found", tagName)
			return
		}

		//fmt.Printf("%+v\n", tag)
		fmt.Printf("Name:     %s\n", tag.Name)
		fmt.Printf("Path:     %s\n", tag.Path)
		fmt.Printf("Location: %s\n", tag.Location)
		fmt.Printf("Revision: %s\n", tag.Revision)
		fmt.Printf("Size:     %d\n", tag.TotalSize)
		fmt.Printf("Created:  %s\n", tag.CreatedAt)
	},
}

func init() {
	regCmd.AddCommand(regTagCmd)
}
