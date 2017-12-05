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
	"unicode"

	"github.com/pkg/errors"
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
		formatChar := "\n"
		if git.IsHub {
			formatChar = ""
		}

		git := git.New()
		git.Stdout = nil
		git.Stderr = nil
		usage, _ := git.CombinedOutput()
		fmt.Printf("%s%sThese GitLab commands are provided by lab:\n%s\n\n", string(usage), formatChar, labUsage(cmd))
	},
}

func trimRightSpace(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}

func rpad(s string, padding int) string {
	template := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(template, s)
}

var templateFuncs = template.FuncMap{
	"trimTrailingWhitespaces": trimRightSpace,
	"rpad": rpad,
}

const labUsageTmpl = `{{range .Commands}}{{if (and (or .IsAvailableCommand (ne .Name "help")) (and (ne .Name "clone") (ne .Name "version")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}`

func labUsage(c *cobra.Command) string {
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

// parseArgsRemote returns the remote and a number if parsed. Many commands
// accept a remote to operate on and number such as a page id
func parseArgsRemote(args []string) (string, int64, error) {
	if len(args) == 2 {
		n, err := strconv.ParseInt(args[1], 0, 64)
		if err != nil {
			return "", 0, err
		}
		ok, err := git.IsRemote(args[0])
		if err != nil {
			return "", 0, err
		} else if !ok {
			return "", 0, errors.Errorf("%s is not a valid remote", args[0])
		}
		return args[0], n, nil
	}
	if len(args) == 1 {
		ok, err := git.IsRemote(args[0])
		if err != nil {
			return "", 0, err
		}
		if ok {
			return args[0], 0, nil
		}
		n, err := strconv.ParseInt(args[0], 0, 64)
		if err == nil {
			return "", n, nil
		}
		return "", 0, errors.Errorf("%s is not a valid remote or number", args[0])
	}
	return "", 0, nil
}

var (
	// Will be updated to upstream in init() if "upstream" remote exists
	forkedFromRemote = "origin"
	// Will be updated to lab.User() in init() if forkedFrom is "origin"
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
	if cmd, _, err := RootCmd.Find(os.Args[1:]); err != nil || cmd.Use == "clone" {
		// Determine if any undefined flags were passed to "clone"
		if cmd.Use == "clone" && len(os.Args) > 2 {
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

		// Passthrough to git for any unrecognised commands
		git := git.New(os.Args[1:]...)
		err = git.Run()
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
	if err := RootCmd.Execute(); err != nil {
		// Execute has already logged the error
		os.Exit(1)
	}
}
