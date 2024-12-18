package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitconfig "github.com/tcnksm/go-gitconfig"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
	"github.com/zaquestion/lab/internal/logger"
)

// Get internal lab logger instance
var log = logger.GetInstance()

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "lab",
	Short: "lab: A GitLab Command Line Interface Utility",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if ok, err := cmd.Flags().GetBool("version"); err == nil && ok {
			versionCmd.Run(cmd, args)
			return
		}
		helpCmd.Run(cmd, args)
	},
}

func rpad(s string, padding int) string {
	template := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(template, s)
}

var templateFuncs = template.FuncMap{
	"rpad": rpad,
}

const labUsageTmpl = `{{range .Commands}}{{if (and (not .Hidden) (or .IsAvailableCommand (ne .Name "help")) (and (ne .Name "clone") (ne .Name "version") (ne .Name "merge-request")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}`

func labUsageFormat(c *cobra.Command) string {
	t := template.New("top")
	t.Funcs(templateFuncs)
	template.Must(t.Parse(labUsageTmpl))

	var buf bytes.Buffer
	err := t.Execute(&buf, c)
	if err != nil {
		c.Println(err)
	}
	return buf.String()
}

func helpFunc(cmd *cobra.Command, args []string) {
	// When help func is called from the help command args will be
	// populated. When help is called with cmd.Help(), the args are not
	// passed through, so we pick them up ourselves here
	if len(args) == 0 {
		args = os.Args[1:]
	}
	rootCmd := cmd.Root()
	// Show help for sub/commands -- any commands that isn't "lab" or "help"
	if cmd, _, err := rootCmd.Find(args); err == nil {
		// Cobra will check parent commands for a helpFunc and we only
		// want the root command to actually use this custom help func.
		// Here we trick cobra into thinking that there is no help func
		// so it will use the default help for the subcommands
		cmd.Root().SetHelpFunc(nil)
		err2 := cmd.Help()
		if err2 != nil {
			log.Fatal(err)
		}
		return
	}
}

var helpCmd = &cobra.Command{
	Use:   "help [command [subcommand...]]",
	Short: "Show the help for lab",
	Long:  ``,
	Run:   helpFunc,
}

// Version is set with linker flags during build.
var Version string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s %s\n", "lab version", Version)
	},
}

func init() {
	// NOTE: Calling SetHelpCommand like this causes helpFunc to be called
	// with correct arguments. If the default cobra help func is used no
	// arguments are passed through and subcommand help breaks.
	RootCmd.SetHelpCommand(helpCmd)
	RootCmd.SetHelpFunc(helpFunc)
	RootCmd.AddCommand(versionCmd)
	RootCmd.Flags().Bool("version", false, "Show the lab version")
	RootCmd.PersistentFlags().Bool("no-pager", false, "Do not pipe output into a pager")
	RootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging level")
	RootCmd.PersistentFlags().Bool("quiet", false, "Turn off any sort of logging. Only command output is printed")

	// We need to set the logger level before any other piece of code is
	// called, thus we make sure we don't lose any debug message, but for
	// that we need to parse the args from command input and let flag errors be
	// handled by the subcommands themselves.
	_ = RootCmd.ParseFlags(os.Args[1:])
	debugLogger, _ := RootCmd.Flags().GetBool("debug")
	quietLogger, _ := RootCmd.Flags().GetBool("quiet")
	if debugLogger && quietLogger {
		log.Fatal("option --debug cannot be combined with --quiet")
	}
	if debugLogger {
		log.SetLogLevel(logger.LogLevelDebug)
	} else if quietLogger {
		log.SetLogLevel(logger.LogLevelNone)
	}
	carapace.Gen(RootCmd)
}

var (
	// Will be updated to upstream in Execute() if "upstream" remote exists
	defaultRemote = ""
	// Will be updated to lab.User() in Execute() if forkedFrom is "origin"
	forkRemote = ""
)

// Try to guess what should be the default remote.
func guessDefaultRemote() string {
	// Allow to force a default remote. If set, return early.
	if config := getMainConfig(); config != nil {
		defaultRemote := config.GetString("core.default_remote")
		if defaultRemote != "" {
			return defaultRemote
		}
		remoteFromBranch := config.GetBool("core.remote_from_branch")
		currentBranch, err := git.CurrentBranch()
		if remoteFromBranch && err == nil {
			remoteConf := fmt.Sprintf("branch.%s.remote", currentBranch)
			if remote, err := gitconfig.Local(remoteConf); err == nil {
				return remote
			}
		}
	}

	guess := ""

	// defaultRemote should try to always point to the upstream project.
	// Since "origin" may have two different meanings depending on how the
	// user forked the project, thus make "upstream" as the most significant
	// remote.
	// In forkRemoteProject approach, "origin" remote is the one pointing to
	// the upstream project by default.
	_, err := gitconfig.Local("remote.origin.url")
	if err == nil {
		guess = "origin"
	}
	// In forkCleanProject approach, "upstream" remote is the one pointing
	// to the upstream project by default.
	_, err = gitconfig.Local("remote.upstream.url")
	if err == nil {
		guess = "upstream"
	}

	// But it's still possible the user used a custom name
	if guess == "" {
		// use the remote tracked by the default branch if set
		if remote, err := gitconfig.Local("branch.main.remote"); err == nil {
			guess = remote
		} else if remote, err = gitconfig.Local("branch.master.remote"); err == nil {
			guess = remote
		} else {
			// use the first remote added to .git/config file, which, usually, is
			// the one from which the repo was clonned
			remotesStr, err := git.GetLocalRemotesFromFile()
			if err == nil {
				remotes := strings.Split(remotesStr, "\n")
				// remotes format: remote.<name>.<url|fetch>
				remoteName := strings.Split(remotes[0], ".")[1]
				guess = remoteName
			}
		}
	}

	return guess
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(initSkipped bool) {
	// Try to gather remote information if running inside a git tree/repo.
	// Otherwise, skip it, since the info won't be used at all, also avoiding
	// misleading error/warning messages about missing remote.
	if !initSkipped && git.InsideGitRepo() {
		defaultRemote = guessDefaultRemote()
		if defaultRemote == "" {
			log.Infoln("No default remote found")
		}

		// Check if the user fork exists
		_, err := gitconfig.Local("remote." + lab.User() + ".url")
		if err == nil {
			forkRemote = lab.User()
		} else {
			forkRemote = defaultRemote
		}
	}

	// Set commandPrefix
	cmd, _, _ := RootCmd.Find(os.Args[1:])
	scmd, _, _ := cmd.Find(os.Args)
	setCommandPrefix(scmd)

	if err := RootCmd.Execute(); err != nil {
		// Execute has already logged the error
		os.Exit(1)
	}
}
