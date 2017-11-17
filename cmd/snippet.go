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

// snippetCmd represents the snippet command
var snippetCmd = &cobra.Command{
	Use:     "snippet",
	Aliases: []string{"snip"},
	Short:   "Create a snippet on GitLab or in a project",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		rn, err := git.PathWithNameSpace("origin")
		if err != nil {
			log.Fatal(err)
		}
		msgs, err := cmd.Flags().GetStringSlice("message")
		if err != nil {
			log.Fatal(err)
		}
		path, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatal(err)
		}
		code, err := determineContents(path)
		if err != nil {
			log.Fatal(err)
		}
		title, _, err := determineMsg(msgs, code)
		if title == "" {
			log.Fatal("aborting snippet due to empty msg")
		}

		name, err := cmd.Flags().GetString("name")
		if err != nil {
			log.Fatal(err)
		}
		visibility := gitlab.InternalVisibility
		if ok, err := cmd.Flags().GetBool("private"); err != nil && ok {
			visibility = gitlab.PrivateVisibility
		} else if ok, err := cmd.Flags().GetBool("public"); err != nil && ok {
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

	title, body, err := git.Edit("SNIPPET", string(b))
	if err != nil {
		_, f, l, _ := runtime.Caller(0)
		log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
	}
	return title, body, err
}

func snippetMsg(initMsg string) (string, error) {
	const tmpl = `{{.InitMsg}}
{{.CommentChar}} Write a message for this snippet. The first block
{{.CommentChar}} of text is the title and the rest is the description.`

	commentChar := git.CommentChar()

	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return "", err
	}

	msg := &struct {
		InitMsg     string
		CommentChar string
	}{
		InitMsg:     initMsg,
		CommentChar: commentChar,
	}

	var b bytes.Buffer
	err = t.Execute(&b, msg)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

func determineContents(path string) (string, error) {
	b, err := ioutil.ReadFile(path)
	if !os.IsNotExist(err) && err != nil {
		return "", err
	}
	if len(b) > 0 {
		return string(b), nil
	}

	if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) != 0 {
		// nothing to read on stdin
		return "", nil
	}
	b, err = ioutil.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func init() {
	snippetCmd.Flags().Bool("private", false, "Make snippet private; visible only to project members (default: internal)")
	snippetCmd.Flags().Bool("public", false, "Make snippet public; can be accessed without any authentication (default: internal)")
	snippetCmd.Flags().StringP("name", "n", "", "(optional) name snippet to add code highlighting, e.g. potato.go for GoLang")
	snippetCmd.Flags().StringP("file", "f", "", "Use the given file to load the contents of the snippet")
	snippetCmd.Flags().StringSliceP("message", "m", []string{}, "Use the given file <msg>; multiple -m are concatenated as seperate paragraphs")
	RootCmd.AddCommand(snippetCmd)
}
