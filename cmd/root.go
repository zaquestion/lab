package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"text/template"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	gitconfig "github.com/tcnksm/go-gitconfig"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:                   "lab",
	Short:                 "A Git Wrapper for GitLab",
	Long:                  ``,
	ZshCompletionFunction: zshCompletionFunction,
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

const labUsageTmpl = `{{range .Commands}}{{if (and (or .IsAvailableCommand (ne .Name "help")) (and (ne .Name "clone") (ne .Name "version") (ne .Name "merge-request")))}}
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
	if cmd, _, err := rootCmd.Find(args); err == nil &&
		cmd != rootCmd && strings.Split(cmd.Use, " ")[0] != "help" {
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

// parseArgsStr returns a string and a number if parsed. Many commands accept a
// string to operate on (remote or search) and number such as a page id
func parseArgsStr(args []string) (string, int64, error) {
	return parseArgsStringInt(args)
}

// parseArgsStringInt returns a string and a number if parsed.
func parseArgsStringInt(args []string) (string, int64, error) {
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

// parseArgs returns a remote name and a number if parsed
func parseArgs(args []string) (string, int64, error) {
	return parseArgsRemoteInt(args)
}

// parseArgsRemoteInt is similar to parseArgsStringInt except that it uses the
// string argument as a remote and returns the project name for that remote
func parseArgsRemoteInt(args []string) (string, int64, error) {
	if !git.InsideGitRepo() {
		return "", 0, nil
	}
	remote, num, err := parseArgsStr(args)
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

// parseArgsRemoteString returns a remote name and a string if parsed.
// If there is an error, it returns two empty strings.
// If no remote is given, it returns the project name of the default remote
// (ie 'origin').
// If no second argument is given, it returns "" as second return value.
func parseArgsRemoteString(args []string) (string, string, error) {
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

var (
	// Will be updated to upstream in Execute() if "upstream" remote exists
	forkedFromRemote = "origin"
	// Will be updated to lab.User() in Execute() if forkedFrom is "origin"
	forkRemote = "origin"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	_, err := gitconfig.Local("remote.upstream.url")
	if err == nil {
		forkedFromRemote = "upstream"
	}

	if forkedFromRemote == "origin" {
		// Check if the user fork exists
		_, err = gitconfig.Local("remote." + lab.User() + ".url")
		if err == nil {
			forkRemote = lab.User()
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
		var hFlaged bool
		for _, v := range os.Args {
			if v == "--help" {
				hFlaged = true
			}
		}
		if !hFlaged {
			git.New(os.Args[1:]...).Run()
			return
		}
	}
	if err := RootCmd.Execute(); err != nil {
		// Execute has already logged the error
		os.Exit(1)
	}
}
