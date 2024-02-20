// This file contains common functions that are shared in the lab package

package cmd

import (
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	gitconfig "github.com/tcnksm/go-gitconfig"
	giturls "github.com/whilp/git-urls"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/config"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	commandPrefix string
	// http vs ssh protocol control flag
	useHTTP bool
)

// flagConfig compares command line flags and the flags set in the config
// files.  The command line value will always override any value set in the
// config files.
func flagConfig(fs *flag.FlagSet) {
	var cmdFlags string

	fs.VisitAll(func(f *flag.Flag) {
		var (
			configValue  interface{}
			configString string
			flagChanged  bool
		)

		switch f.Value.Type() {
		case "bool":
			configValue = getMainConfig().GetBool(commandPrefix + f.Name)
			configString = strconv.FormatBool(configValue.(bool))
		case "string":
			configValue = getMainConfig().GetString(commandPrefix + f.Name)
			configString = configValue.(string)
		case "stringSlice":
			configValue = getMainConfig().GetStringSlice(commandPrefix + f.Name)
			configString = strings.Join(configValue.([]string), " ")
		case "int":
			log.Fatal("ERROR: found int flag, use string instead: ", f.Value.Type(), f)
		case "stringArray":
			// viper does not have support for stringArray
			configString = ""
		default:
			log.Fatal("ERROR: found unidentified flag: ", f.Value.Type(), f)
		}

		if f.Changed {
			flagChanged = true
		}

		// o/w use the value in the configfile
		if !flagChanged && configString != "" && configString != f.DefValue {
			f.Value.Set(configString)
			flagChanged = true
		}

		if flagChanged {
			if f.Name != "debug" {
				cmdFlags += fmt.Sprintf("  %s = %s\n", f.Name, f.Value.String())
			}
		}
	})

	if len(cmdFlags) != 0 {
		log.Debugf("command flags enabled: \n%s", cmdFlags)
	}
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
	if err != nil {
		// Fall back to local branch
		mrBranch = branch
	}

	branchRemote, err := determineSourceRemote(branch)
	if err != nil {
		log.Fatal(err)
	}

	branchProjectName, err := git.PathWithNamespace(branchRemote)
	if err != nil {
		log.Fatal(err)
	}

	branchProject, err := lab.FindProject(branchProjectName)
	if err != nil {
		log.Fatal(err)
	}

	mrs, err := lab.MRList(rn, gitlab.ListProjectMergeRequestsOptions{
		State:        &mrState,
		OrderBy:      gitlab.String("updated_at"),
		SourceBranch: gitlab.String(mrBranch),
	}, -1)
	if err != nil {
		log.Fatal(err)
	}

	for _, mr := range mrs {
		if mr.SourceProjectID == branchProject.ID {
			num = mr.IID
			break
		}
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
	rn, err := git.PathWithNamespace(remote)
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
	if err != nil {
		return "", "", err
	} else if branch == "" {
		branch, err = git.CurrentBranch()
	}

	remoteBranch, _ := git.UpstreamBranch(branch)
	if remoteBranch != "" {
		branch = remoteBranch
	}

	if remote == "" {
		remote, err = determineSourceRemote(branch)
		if err != nil {
			return "", "", err
		}
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

		// In this part, we only really care about the latest pipeline that
		// ran, regardless its result.
		if mr.HeadPipeline == nil {
			return "", 0, errors.Errorf("No pipeline found for merge request %d", mrNum)
		}

		// MR pipelines may run on the source, target or another project
		// (multi-project pipelines), and we don't have a proper way to
		// know which it is. Here we handle the first two cases.
		if strings.Contains(mr.HeadPipeline.WebURL, rn) {
			return rn, mr.HeadPipeline.ID, nil
		}

		p, err := lab.GetProject(mr.SourceProjectID)
		if err != nil {
			return "", 0, err
		}

		return p.PathWithNamespace, mr.HeadPipeline.ID, nil
	}
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

func getRemoteName(remote string) (string, error) {
	if remote == "" {
		remote = defaultRemote
	}

	ok, err := git.IsRemote(remote)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.Errorf("%s is not a valid remote", remote)
	}

	remote, err = git.PathWithNamespace(remote)
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
	rn, id, err := parseArgsRemoteAndID(args)
	if err == nil && id != 0 {
		return rn, id, nil
	}

	rn, branch, err := parseArgsRemoteAndString(args)
	if err != nil {
		return "", 0, err
	}

	rn, err = getRemoteName(rn)
	if err != nil {
		return "", 0, err
	}

	if branch == "" {
		id = int64(getCurrentBranchMR(rn))
	} else {
		id = int64(getBranchMR(rn, branch))
	}

	if id == 0 {
		err = fmt.Errorf("cannot determine MR id")
		return "", 0, err
	}

	return rn, id, nil
}

// filterCommentArg separate the case where a command can have both the
// remote and "<mrID>:<commentID>" at the same time.
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
	} else if len(args) == 2 {
		branchArgs = append(branchArgs, args[0])
		idString = args[1]
	}

	if strings.Contains(idString, ":") {
		ps := strings.Split(idString, ":")
		branchArgs = append(branchArgs, ps[0])
		idString = ps[1]
	} else if idString != "" {
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
		if commandPrefix != "" {
			break
		}
		commandName := strings.Split(command.Use, " ")[0]
		if scmd == command {
			commandPrefix = commandName + "."
			break
		}
		for _, subcommand := range command.Commands() {
			subCommandName := commandName + "_" + strings.Split(subcommand.Use, " ")[0]
			if scmd == subcommand {
				commandPrefix = subCommandName + "."
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
	if !terminal.IsTerminal(sysStdout) ||
		!terminal.IsTerminal(sysStderr) {
		return false
	}
	return true
}

type pager struct {
	proc   *os.Process
	stdout int
}

// If standard output is a terminal, redirect output to an external
// pager until the returned object's Close() method is called
func newPager(fs *flag.FlagSet) *pager {
	cmdLine, env := git.PagerCommand()
	args := strings.Split(cmdLine, " ")

	noPager, _ := fs.GetBool("no-pager")
	if !isOutputTerminal() || noPager || args[0] == "cat" {
		return &pager{}
	}

	pr, pw, _ := os.Pipe()
	defer pw.Close()

	name, _ := exec.LookPath(args[0])
	proc, _ := os.StartProcess(name, args, &os.ProcAttr{
		Env:   env,
		Files: []*os.File{pr, os.Stdout, os.Stderr},
	})

	savedStdout, _ := dupFD(sysStdout)
	_ = dupFD2(int(pw.Fd()), sysStdout)

	return &pager{
		proc:   proc,
		stdout: savedStdout,
	}
}

// Close closes the pager
func (p *pager) Close() {
	if p.stdout > 0 {
		_ = dupFD2(p.stdout, sysStdout)
		_ = closeFD(p.stdout)
	}
	if p.proc != nil {
		p.proc.Wait()
	}
}

func labPersistentPreRun(cmd *cobra.Command, args []string) {
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

func determineSourceRemote(branch string) (string, error) {
	// There is a precendence of options that should be considered here:
	// branch.<name>.pushRemote > remote.pushDefault > branch.<name>.remote
	// This rule is placed in git-config(1) manpage
	r, err := gitconfig.Local("branch." + branch + ".pushRemote")
	if err != nil {
		r, err = gitconfig.Local("remote.pushDefault")
		if err != nil {
			r, err = gitconfig.Local("branch." + branch + ".remote")
			if err != nil {
				return forkRemote, nil
			}
		}
	}

	// Parse the remote name for possible URL.
	u, err := giturls.Parse(r)
	if err != nil {
		return "", err
	}

	path := strings.TrimPrefix(u.Path, "/")
	return path, nil
}

// Check of a case-insensitive prefix in a string
func hasPrefix(str, prefix string) bool {
	if len(str) < len(prefix) {
		return false
	}
	return strings.EqualFold(str[0:len(prefix)], prefix)
}

// Match terms being searched with an existing list of terms, checking its
// ambiguity at the same time
func matchTerms(searchTerms, existentTerms []string) ([]string, error) {
	var ambiguous bool
	matches := make([]string, len(searchTerms))

	for i, sTerm := range searchTerms {
		ambiguous = false
		lowerSTerm := strings.ToLower(sTerm)
		for _, eTerm := range existentTerms {
			lowerETerm := strings.ToLower(eTerm)

			// no match
			if !strings.Contains(lowerETerm, lowerSTerm) {
				continue
			}

			// check for ambiguity on substring level
			if matches[i] != "" && lowerSTerm != lowerETerm {
				ambiguous = true
				continue
			}

			matches[i] = eTerm

			// exact match
			// may happen after multiple substring matches
			if lowerETerm == lowerSTerm {
				ambiguous = false
				break
			}
		}

		if matches[i] == "" {
			return nil, errors.Errorf("'%s' not found", sTerm)
		}

		// Ambiguous matches should not be returned to avoid
		// manipulating the wrong item.
		if ambiguous {
			return nil, errors.Errorf("'%s' has no exact match and is ambiguous", sTerm)
		}
	}

	return matches, nil
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

// getUser returns the userID for use with other GitLab API calls.
func getUserID(user string) *int {
	var (
		err    error
		userID int
	)

	if user == "" {
		return nil
	}

	if user[0] == '@' {
		user = user[1:]
	}

	if strings.Contains(user, "@") {
		userID, err = lab.UserIDFromEmail(user)
	} else {
		userID, err = lab.UserIDFromUsername(user)
	}
	if err != nil {
		return nil
	}
	if userID == -1 {
		return nil
	}

	return gitlab.Int(userID)
}

// getUsers returns the userIDs for use with other GitLab API calls.
func getUserIDs(users []string) []int {
	var ids []int
	for _, user := range users {
		userID := getUserID(user)
		if userID != nil {
			ids = append(ids, *userID)
		} else {
			fmt.Printf("Warning: %s is not a valid username\n", user)
		}
	}
	return ids
}

// mapLabelsAsStrings returns a list of labels as an array of strings
func mapLabelsAsStrings(rn string, labelTerms []string) ([]string, error) {
	// Don't bother fetching project labels if nothing is being really requested
	if len(labelTerms) == 0 {
		return []string{}, nil
	}

	labels, err := lab.LabelList(rn)
	if err != nil {
		return nil, err
	}

	labelNames := make([]string, len(labels))
	for _, label := range labels {
		labelNames = append(labelNames, label.Name)
	}

	matches, err := matchTerms(labelTerms, labelNames)
	if err != nil {
		return nil, errors.Errorf("Label %s\n", err.Error())
	}

	return matches, nil
}

// malLabelsAsLabels returns a list of labels as gitlab.LabelOptions
func mapLabelsAsLabelOptions(rn string, labelTerms []string) (gitlab.LabelOptions, error) {

	matches, err := mapLabelsAsStrings(rn, labelTerms)
	if err != nil {
		return nil, err
	}

	return gitlab.LabelOptions(matches), nil
}

// dumpToken dumps information about a specific Personal Access Token
func dumpToken(tokendata *gitlab.PersonalAccessToken) {
	fmt.Println("ID:        ", tokendata.ID)
	fmt.Println("Name:      ", tokendata.Name)
	fmt.Println("Revoked:   ", tokendata.Revoked)
	fmt.Println("CreatedAt: ", tokendata.CreatedAt)
	fmt.Println("Scopes:    ", strings.Join(tokendata.Scopes, ","))
	fmt.Println("UserID:    ", tokendata.UserID)
	fmt.Println("LastUsedAt:", tokendata.LastUsedAt)
	fmt.Println("Active:    ", tokendata.Active)
	fmt.Println("ExpiresAt: ", time.Time(*tokendata.ExpiresAt).String())
	fmt.Println("")
}

// HexToRGB converts hex color to color.RGBA with "#FFFFFF" format
func HexToRGB(hex string) color.RGBA {
	values, _ := strconv.ParseUint(string(hex[1:]), 16, 32)
	return color.RGBA{R: uint8(values >> 16), G: uint8((values >> 8) & 0xFF), B: uint8(values & 0xFF), A: 255}
}
