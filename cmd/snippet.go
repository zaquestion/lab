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
	path    string
	name    string
	private bool
	public  bool
)

// snippetCmd represents the snippet command
var snippetCmd = &cobra.Command{
	Use:     "snippet",
	Aliases: []string{"snip"},
	Short:   "Create a snippet on GitLab or in a project",
	Long: `
Source snippets from stdin, file, or in editor from scratch
Write title&description in editor, or -m`,
	Run: func(cmd *cobra.Command, args []string) {
		rn, err := git.PathWithNameSpace(forkedFromRemote)
		if err != nil {
			log.Fatal(err)
		}
		code, err := determineCode(path)
		if err != nil {
			log.Fatal(err)
		}
		if code == "" {
			log.Fatal("aborting snippet due to empty contents")
		}
		title, _, err := determineMsg(msgs, code)
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
		// TODO: expand gitlab api to support creating snippets with descriptions
		snip, err := lab.CreateSnippet(rn, &gitlab.CreateSnippetOptions{
			Title:      gitlab.String(title),
			Code:       gitlab.String(code),
			FileName:   gitlab.String(name),
			Visibility: &visibility,
		})
		if err != nil {
			log.Fatal(err)
		}
		if snip == nil {
			log.Fatal("Fatal: snippet failed to be created")
		}
		// TODO: expand gitlab api to expose web_url field
		// https://github.com/xanzy/go-gitlab/pull/247
		project := lab.User() + "/" + rn
		if strings.Contains(rn, "/") {
			project = rn
		}
		fmt.Printf("%s/%s/%d", lab.Host(), project, snip.ID)
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
	snippetCmd.Flags().BoolVarP(&private, "private", "p", false, "Make snippet private; visible only to project members (default: internal)")
	snippetCmd.Flags().BoolVar(&public, "public", false, "Make snippet public; can be accessed without any authentication (default: internal)")
	snippetCmd.Flags().StringVarP(&name, "name", "n", "", "(optional) Name snippet to add code highlighting, e.g. potato.go for GoLang")
	snippetCmd.Flags().StringSliceVarP(&msgs, "message", "m", []string{}, "Use the given file <msg>; multiple -m are concatenated as seperate paragraphs")
	RootCmd.AddCommand(snippetCmd)
}
