// This file contains common functions that are shared in the lab package
package cmd

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	gitconfig "github.com/tcnksm/go-gitconfig"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/config"
	git "github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	CommandPrefix string
	// http vs ssh protocol control flag
	useHTTP bool
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
		if f.Changed {
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
	currentBranch, err := git.CurrentBranch()
	if err != nil {
		return 0
	}

	return getBranchMR(rn, currentBranch)
}

func getBranchMR(rn, branch string) int {
	var num int = 0

	mrBranch, err := git.UpstreamBranch(branch)
	if mrBranch == "" {
		// Fall back to local branch
		mrBranch = branch
	}

	mrs, err := lab.MRList(rn, gitlab.ListProjectMergeRequestsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 10,
		},
		Labels:       mrLabels,
		State:        &mrState,
		OrderBy:      gitlab.String("updated_at"),
		SourceBranch: gitlab.String(mrBranch),
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
		remote = defaultRemote
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

	remote, str, err := parseArgsRemoteAndString(args)
	if err != nil {
		return "", "", nil
	}

	if remote == "" {
		remote = defaultRemote
	}

	remote, err = getRemoteName(remote)
	if err != nil {
		return "", "", err
	}
	return remote, str, nil
}

// parseArgsRemoteAndBranch is used by commands to parse command line
// arguments.  This function returns a remote name and a branch name.
// If no branch name is given, the function returns the upstream of
// the current branch and the corresponding remote.
func parseArgsRemoteAndBranch(args []string) (string, string, error) {
	if !git.InsideGitRepo() {
		return "", "", nil
	}

	remote, branch, err := parseArgsRemoteAndString(args)
	if branch == "" && err == nil {
		branch, err = git.CurrentBranch()
	}

	if err != nil {
		return "", "", err
	}

	if remote == "" {
		remote = determineSourceRemote(branch)
	}

	remoteBranch, _ := git.UpstreamBranch(branch)
	if remoteBranch != "" {
		branch = remoteBranch
	}

	remote, err = getRemoteName(remote)
	if err != nil {
		return "", "", err
	}
	return remote, branch, nil
}

func getPipelineFromArgs(args []string, forMR bool) (string, int, error) {
	if forMR {
		rn, mrNum, err := parseArgsWithGitBranchMR(args)
		if err != nil {
			return "", 0, err
		}

		mr, err := lab.MRGet(rn, int(mrNum))
		if err != nil {
			return "", 0, err
		}

		if mr.Pipeline == nil {
			return "", 0, errors.Errorf("No pipeline found for merge request %d", mrNum)
		}

		// MR pipelines may run on the source- or target project,
		// and we don't have a proper way to know which it is
		if strings.Contains(mr.Pipeline.WebURL, rn) {
			return rn, mr.Pipeline.ID, nil
		} else {
			p, err := lab.GetProject(mr.SourceProjectID)
			if err != nil {
				return "", 0, err
			}

			return p.PathWithNamespace, mr.Pipeline.ID, nil
		}
	} else {
		rn, refName, err := parseArgsRemoteAndBranch(args)
		if err != nil {
			return "", 0, err
		}

		commit, err := lab.GetCommit(rn, refName)
		if err != nil {
			return "", 0, err
		}

		if commit.LastPipeline == nil {
			return "", 0, errors.Errorf("No pipeline found for %s", refName)
		}

		return rn, commit.LastPipeline.ID, nil
	}
}

func getRemoteName(remote string) (string, error) {
	ok, err := git.IsRemote(remote)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.Errorf("%s is not a valid remote", remote)
	}

	remote, err = git.PathWithNameSpace(remote)
	if err != nil {
		return "", err
	}

	return remote, nil
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

func parseArgsRemoteAndString(args []string) (string, string, error) {
	remote, str := "", ""

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

	return remote, str, nil
}

// parseArgsWithGitBranchMR returns a remote name and a number if parsed.
// If no number is specified, the MR id associated with the given branch
// is returned, using the current branch as fallback.
func parseArgsWithGitBranchMR(args []string) (string, int64, error) {
	var (
		s      string
		branch string
		err    error
	)
	s, i, err := parseArgsRemoteAndID(args)
	if i == 0 {
		s, branch, err = parseArgsRemoteAndString(args)
		if err != nil {
			return "", 0, err
		}

		if s == "" {
			s = defaultRemote
		}
		s, err = getRemoteName(s)
		if err != nil {
			return "", 0, err
		}

		if branch == "" {
			i = int64(getCurrentBranchMR(s))
		} else {
			i = int64(getBranchMR(s, branch))
		}
		if i == 0 {
			fmt.Println("Error: Cannot determine MR id.")
			os.Exit(1)
		}
	}
	return s, i, nil
}

func filterCommentArg(args []string) (int, []string, error) {
	branchArgs := []string{}
	idString := ""

	if len(args) == 1 {
		ok, err := git.IsRemote(args[0])
		if err != nil {
			return 0, branchArgs, err
		}
		if ok {
			branchArgs = append(branchArgs, args[0])
		} else {
			idString = args[0]
		}
	} else if len(args) > 1 {
		branchArgs = append(branchArgs, args[0])
		idString = args[1]
	}

	if strings.Contains(idString, ":") {
		ps := strings.Split(idString, ":")
		branchArgs = append(branchArgs, ps[0])
		idString = ps[1]
	} else {
		branchArgs = append(branchArgs, idString)
		idString = ""
	}

	idNum, _ := strconv.Atoi(idString)
	return idNum, branchArgs, nil
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

// isOutputTerminal checks if both stdout and stderr are indeed terminals
// to avoid some markdown rendering garbage going to other outputs that
// don't support some control chars.
func isOutputTerminal() bool {
	if !terminal.IsTerminal(int(syscall.Stdout)) ||
		!terminal.IsTerminal(int(syscall.Stderr)) {
		return false
	}
	return true
}

func LabPersistentPreRun(cmd *cobra.Command, args []string) {
	flagConfig(cmd.Flags())
}

// labURLToRepo returns the string representing the URL to a certain repo based
// on the protocol used
func labURLToRepo(project *gitlab.Project) string {
	urlToRepo := project.SSHURLToRepo
	if useHTTP {
		urlToRepo = project.HTTPURLToRepo
	}
	return urlToRepo
}

func determineSourceRemote(branch string) string {
	// There is a precendence of options that should be considered here:
	// branch.<name>.pushRemote > remote.pushDefault > branch.<name>.remote
	// This rule is placed in git-config(1) manpage
	r, err := gitconfig.Local("branch." + branch + ".pushRemote")
	if err == nil {
		return r
	}
	r, err = gitconfig.Local("remote.pushDefault")
	if err == nil {
		return r
	}
	r, err = gitconfig.Local("branch." + branch + ".remote")
	if err == nil {
		return r
	}

	return forkRemote
}

// union returns all the unique elements in a and b
func union(a, b []string) []string {
	mb := map[string]bool{}
	ab := []string{}
	for _, x := range b {
		mb[x] = true
		// add all of b's elements to ab
		ab = append(ab, x)
	}
	for _, x := range a {
		if _, ok := mb[x]; !ok {
			// if a's elements aren't in b, add them to ab
			// if they are, we don't need to add them
			ab = append(ab, x)
		}
	}
	return ab
}

// difference returns the elements in a that aren't in b
func difference(a, b []string) []string {
	mb := map[string]bool{}
	for _, x := range b {
		mb[x] = true
	}
	ab := []string{}
	for _, x := range a {
		if _, ok := mb[x]; !ok {
			ab = append(ab, x)
		}
	}
	return ab
}

// same returns true if a and b contain the same strings (regardless of order)
func same(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	mb := map[string]bool{}
	for _, x := range b {
		mb[x] = true
	}

	for _, x := range a {
		if _, ok := mb[x]; !ok {
			return false
		}
	}
	return true
}
