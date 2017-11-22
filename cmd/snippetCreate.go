package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var (
	msgs    []string
	name    string
	file    string
	private bool
	public  bool
)

// mrCmd represents the mr command
var snippetCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a snippet on GitLab or in a project",
	Long: `
Source snippets from stdin, file, or in editor from scratch
Write title & description in editor, or using -m`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			file = args[0]
		}
		code, err := determineCode(file)
		if err != nil {
			log.Fatal(err)
		}
		if strings.TrimSpace(code) == "" {
			log.Fatal("aborting snippet due to empty contents")
		}
		title, body, err := determineMsg(msgs, code)
		if title == "" {
			log.Fatal("aborting snippet due to empty msg")
		}

		visibility := gitlab.InternalVisibility
		switch {
		case private:
			visibility = gitlab.PrivateVisibility
		case public:
			visibility = gitlab.PublicVisibility
		}

		if rn, _ := git.PathWithNameSpace(forkRemote); rn != "" {
			// Looking up the fork ensures there is only 1 api call
			// because it returns the forked from Project
			project, err := lab.FindProject(rn)
			if err != nil {
				log.Fatal(err)
			}
			psOpts := gitlab.CreateProjectSnippetOptions{
				Title:       gitlab.String(title),
				Description: gitlab.String(body),
				Code:        gitlab.String(code),
				FileName:    gitlab.String(name),
				Visibility:  &visibility,
			}
			// Assuming that if you have permissions to create
			// snippets on forkedFromRepo thats what you want
			snip, err := lab.ProjectSnippetCreate(project.ForkedFromProject.ID, &psOpts)
			if err == nil && snip != nil {
				fmt.Println(snip.WebURL)
				return
			}

			// Try creating on user fork if failed to create on
			// forkedFromRepo. Seemingly the next best bet
			if err != nil || snip == nil {
				snip, err = lab.ProjectSnippetCreate(project.ID, &psOpts)
				if err == nil && snip != nil {
					fmt.Println(snip.WebURL)
					return
				}
			}
		}

		sOpts := gitlab.CreateSnippetOptions{
			Title:       gitlab.String(title),
			Description: gitlab.String(body),
			Content:     gitlab.String(code),
			FileName:    gitlab.String(name),
			Visibility:  &visibility,
		}
		snip, err := lab.SnippetCreate(&sOpts)
		if err != nil {
			log.Fatal(err)

		}
		if snip == nil {
			log.Fatal("failed to create snippet")
		}
		fmt.Println(snip.WebURL)
	},
}

func determineMsg(msgs []string, code string) (string, string, error) {
	if len(msgs) > 0 {
		return msgs[0], strings.Join(msgs[1:], "\n\n"), nil
	}

	// Read up to the first 30 chars
	buf := bytes.NewBufferString(code)
	reader := io.LimitReader(buf, 30)
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", "", nil
	}
	i := bytes.IndexByte(b, '\n')
	if i != -1 {
		b = b[:i]
	}

	var tmpl = string(b) + `
{{.CommentChar}} Write a message for this snippet. The first block
{{.CommentChar}} is the title and the rest is the description.`

	msg, err := snipText(tmpl)
	if err != nil {
		log.Fatal(err)
	}

	title, body, err := git.Edit("SNIPMSG", msg)
	if err != nil {
		_, f, l, _ := runtime.Caller(0)
		log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
	}
	return title, body, err
}

func determineCode(path string) (string, error) {
	b, err := ioutil.ReadFile(path)
	if !os.IsNotExist(err) && err != nil {
		return "", err
	}
	if len(b) > 0 {
		return string(b), nil
	}

	if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
		b, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		if len(b) > 0 {
			return string(b), nil
		}
	}

	var tmpl = string(b) + `
{{.CommentChar}} In this mode you are writing a snippet from scratch
{{.CommentChar}} The first block is the title and the rest is the contents.`

	msg, err := snipText(tmpl)
	if err != nil {
		log.Fatal(err)
	}
	title, body, err := git.Edit("SNIPCODE", msg)
	if err != nil {
		_, f, l, _ := runtime.Caller(0)
		log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
	}
	return fmt.Sprintf("%s\n\n%s", title, body), nil
}

func snipText(tmpl string) (string, error) {
	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return "", err
	}

	cc := git.CommentChar()
	msg := &struct{ CommentChar string }{CommentChar: cc}
	var b bytes.Buffer
	err = t.Execute(&b, msg)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

func init() {
	snippetCreateCmd.Flags().BoolVarP(&private, "private", "p", false, "Make snippet private; visible only to project members (default: internal)")
	snippetCreateCmd.Flags().BoolVar(&public, "public", false, "Make snippet public; can be accessed without any authentication (default: internal)")
	snippetCreateCmd.Flags().StringVarP(&name, "name", "n", "", "(optional) Name snippet to add code highlighting, e.g. potato.go for GoLang")
	snippetCreateCmd.Flags().StringSliceVarP(&msgs, "message", "m", []string{}, "Use the given file <msg>; multiple -m are concatenated as seperate paragraphs")
	snippetCmd.Flags().AddFlagSet(snippetCreateCmd.Flags())
	snippetCmd.AddCommand(snippetCreateCmd)
}
