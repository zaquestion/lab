// This file contains common functions that are shared in the lab package
package cmd

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/config"
	git "github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	CommandPrefix string
)

// flagConfig compares command line flags and the flags set in the config
// files.  The command line value will always override any value set in the
// config files.
func flagConfig(fs *flag.FlagSet) {
	fs.VisitAll(func(f *flag.Flag) {
		var (
			configValue  interface{}
			configString string
		)

		switch f.Value.Type() {
		case "bool":
			configValue = getMainConfig().GetBool(CommandPrefix + f.Name)
			configString = strconv.FormatBool(configValue.(bool))
		case "string":
			configValue = getMainConfig().GetString(CommandPrefix + f.Name)
			configString = configValue.(string)
		case "stringSlice":
			configValue = getMainConfig().GetStringSlice(CommandPrefix + f.Name)
			configString = strings.Join(configValue.([]string), " ")

		case "int":
			configValue = getMainConfig().GetInt64(CommandPrefix + f.Name)
			configString = strconv.FormatInt(configValue.(int64), 10)
		case "stringArray":
			// viper does not have support for stringArray
			configString = ""
		default:
			log.Fatal("ERROR: found unidentified flag: ", f.Value.Type(), f)
		}

		// if set, always use the command line option (flag) value
		if f.Value.String() != f.DefValue {
			return
		}
		// o/w use the value in the configfile
		if configString != "" && configString != f.DefValue {
			f.Value.Set(configString)
		}
	})
}

// getCurrentBranchMR returns the MR ID associated with the current branch.
// If a MR ID cannot be found, the function returns 0.
func getCurrentBranchMR(rn string) int {
	var num int = 0

	currentBranch, err := git.CurrentBranch()
	if err != nil {
		log.Fatal(err)
	}

	mrs, err := lab.MRList(rn, gitlab.ListProjectMergeRequestsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 10,
		},
		Labels:       mrLabels,
		State:        &mrState,
		OrderBy:      gitlab.String("updated_at"),
		SourceBranch: gitlab.String(currentBranch),
	}, -1)
	if err != nil {
		log.Fatal(err)
	}

	if len(mrs) > 0 {
		num = mrs[0].IID
	}
	return num
}

// getMainConfig returns the merged config of ~/.config/lab/lab.toml and
// .git/lab/lab.toml
func getMainConfig() *viper.Viper {
	return config.MainConfig
}

// parseArgsRemoteAndID is used by commands to parse command line arguments.
// This function returns a remote name and number.
func parseArgsRemoteAndID(args []string) (string, int64, error) {
	if !git.InsideGitRepo() {
		return "", 0, nil
	}

	remote, num, err := parseArgsStringAndID(args)
	if err != nil {
		return "", 0, err
	}
	ok, err := git.IsRemote(remote)
	if err != nil {
		return "", 0, err
	} else if !ok && remote != "" {
		switch len(args) {
		case 1:
			return "", 0, errors.Errorf("%s is not a valid remote or number", args[0])
		default:
			return "", 0, errors.Errorf("%s is not a valid remote", args[0])
		}
	}
	if remote == "" {
		remote = forkedFromRemote
	}
	rn, err := git.PathWithNameSpace(remote)
	if err != nil {
		return "", 0, err
	}
	return rn, num, nil
}

// parseArgsRemoteAndProject is used by commands to parse command line
// arguments.  This function returns a remote name and the project name.  If no
// remote name is given, the function returns "" and the project name of the
// default remote (ie 'origin').
func parseArgsRemoteAndProject(args []string) (string, string, error) {
	if !git.InsideGitRepo() {
		return "", "", nil
	}

	remote, str := forkedFromRemote, ""

	if len(args) == 1 {
		ok, err := git.IsRemote(args[0])
		if err != nil {
			return "", "", err
		}
		if ok {
			remote = args[0]
		} else {
			str = args[0]
		}
	} else if len(args) > 1 {
		remote, str = args[0], args[1]
	}

	ok, err := git.IsRemote(remote)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", errors.Errorf("%s is not a valid remote", remote)
	}

	remote, err = git.PathWithNameSpace(remote)
	if err != nil {
		return "", "", err
	}
	return remote, str, nil
}

// parseArgsStringAndID is used by commands to parse command line arguments.
// This function returns a string and number.
func parseArgsStringAndID(args []string) (string, int64, error) {
	if len(args) == 2 {
		n, err := strconv.ParseInt(args[1], 0, 64)
		if err != nil {
			return args[0], 0, err
		}
		return args[0], n, nil
	}
	if len(args) == 1 {
		n, err := strconv.ParseInt(args[0], 0, 64)
		if err != nil {
			return args[0], 0, nil
		}
		return "", n, nil
	}
	return "", 0, nil
}

// parseArgsWithGitBranchMR returns a remote name and a number if parsed.
// If no number is specified, the MR id associated with the current
// branch is returned.
func parseArgsWithGitBranchMR(args []string) (string, int64, error) {
	s, i, err := parseArgsRemoteAndID(args)
	if i == 0 {
		i = int64(getCurrentBranchMR(s))
		if i == 0 {
			fmt.Println("Error: Cannot determine MR id.")
			os.Exit(1)
		}
	}
	return s, i, err
}

// setCommandPrefix returns a concatenated value of some of the commandline.
// For example, 'lab mr show' would return 'mr_show.', and 'lab issue list'
// would return 'issue_list.'
func setCommandPrefix(scmd *cobra.Command) {
	for _, command := range RootCmd.Commands() {
		if CommandPrefix != "" {
			break
		}
		commandName := strings.Split(command.Use, " ")[0]
		if scmd == command {
			CommandPrefix = commandName + "."
			break
		}
		for _, subcommand := range command.Commands() {
			subCommandName := commandName + "_" + strings.Split(subcommand.Use, " ")[0]
			if scmd == subcommand {
				CommandPrefix = subCommandName + "."
				break
			}
		}
	}
}

// textToMarkdown converts text with markdown friendly line breaks
// See https://gist.github.com/shaunlebron/746476e6e7a4d698b373 for more info.
func textToMarkdown(text string) string {
	text = strings.Replace(text, "\n", "  \n", -1)
	return text
}

func LabPersistentPreRun(cmd *cobra.Command, args []string) {
	flagConfig(cmd.Flags())
}
