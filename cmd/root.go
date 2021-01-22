package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"text/template"

	"github.com/spf13/cobra"
	gitconfig "github.com/tcnksm/go-gitconfig"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "lab",
	Short: "A Git Wrapper for GitLab",
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
	formatChar := "\n"
	if git.IsHub {
		formatChar = ""
	}

	git := git.New()
	git.Stdout = nil
	git.Stderr = nil
	usage, _ := git.CombinedOutput()
	fmt.Printf("%s%sThese GitLab commands are provided by lab:\n%s\n\n", string(usage), formatChar, labUsageFormat(cmd.Root()))
}

var helpCmd = &cobra.Command{
	Use:   "help [command [subcommand...]]",
	Short: "Show the help for lab",
	Long:  ``,
	Run:   helpFunc,
}

func init() {
	// NOTE: Calling SetHelpCommand like this causes helpFunc to be called
	// with correct arguments. If the default cobra help func is used no
	// arguments are passed through and subcommand help breaks.
	RootCmd.SetHelpCommand(helpCmd)
	RootCmd.SetHelpFunc(helpFunc)
	RootCmd.Flags().Bool("version", false, "Show the lab version")
}

var (
	// Will be updated to upstream in Execute() if "upstream" remote exists
	defaultRemote = ""
	// Will be updated to lab.User() in Execute() if forkedFrom is "origin"
	forkRemote = ""
)

// Try to guess what should be the default remote.
func guessDefaultRemote() string {
	guess := ""

	_, err := gitconfig.Local("remote.upstream.url")
	if err == nil {
		guess = "upstream"
	}
	_, err = gitconfig.Local("remote.origin.url")
	if err == nil {
		guess = "origin"
	}

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
func Execute() {
	// Try to gather remote information if running inside a git tree/repo.
	// Otherwise, skip it, since the info won't be used at all, also avoiding
	// misleading error/warning messages about missing remote.
	if git.InsideGitRepo() {
		defaultRemote = guessDefaultRemote()
		if defaultRemote == "" {
			log.Println("No default remote found")
		}

		// Check if the user fork exists
		_, err := gitconfig.Local("remote." + lab.User() + ".url")
		if err == nil {
			forkRemote = lab.User()
		} else {
			forkRemote = defaultRemote
		}
	}

	// Check if the user is calling a lab command or if we should passthrough
	// NOTE: The help command won't be found by Find, which we are counting on
	cmd, _, err := RootCmd.Find(os.Args[1:])
	if err != nil || cmd.Use == "clone" {
		// Determine if any undefined flags were passed to "clone"
		// TODO: Evaluate and support some of these flags
		// NOTE: `hub help -a` wraps the `git help -a` output
		if (cmd.Use == "clone" && len(os.Args) > 2) || os.Args[1] == "help" {
			// ParseFlags will err in these cases
			err = cmd.ParseFlags(os.Args[1:])
			if err == nil {
				if err := RootCmd.Execute(); err != nil {
					// Execute has already logged the error
					os.Exit(1)
				}
				return
			}
		}

		// Lab passthrough for these commands can cause confusion. See #163
		if os.Args[1] == "create" {
			log.Fatalf("Please call `hub create` directly for github, the lab equivalent is `lab project create`")
		}
		if os.Args[1] == "browse" {
			log.Fatalf("Please call `hub browse` directly for github, the lab equivalent is `lab <object> browse`")
		}
		if os.Args[1] == "alias" {
			log.Fatalf("Please call `hub alias` directly for github, there is no lab equivalent`")
		}

		// Passthrough to git for any unrecognized commands
		err = git.New(os.Args[1:]...).Run()
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	// allow flags to the root cmd to be passed through. Technically we'll drop any exit code info which isn't ideal.
	// TODO: remove for 1.0 when we stop wrapping git
	if cmd.Use == RootCmd.Use && len(os.Args) > 1 {
		var knownFlag bool
		for _, v := range os.Args {
			if v == "-h" || v == "--help" || v == "--version" {
				knownFlag = true
			}
		}
		if !knownFlag {
			git.New(os.Args[1:]...).Run()
			return
		}
	}

	// Set CommandPrefix
	scmd, _, _ := cmd.Find(os.Args)
	setCommandPrefix(scmd)

	if err := RootCmd.Execute(); err != nil {
		// Execute has already logged the error
		os.Exit(1)
	}
}
