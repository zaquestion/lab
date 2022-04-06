package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTitleBody(t *testing.T) {
	tests := []struct {
		Name          string
		Message       string
		ExpectedTitle string
		ExpectedBody  string
	}{
		{
			Name:          "Title Only",
			Message:       "test commit",
			ExpectedTitle: "test commit",
			ExpectedBody:  "",
		},
		{
			Name:          "Title and Body",
			Message:       "test commit\n\ntest body",
			ExpectedTitle: "test commit",
			ExpectedBody:  "test body",
		},
		{
			Name:          "Title and Body mixed comments",
			Message:       "test commit\n\ntest body\n# comments\nmore of body",
			ExpectedTitle: "test commit",
			ExpectedBody:  "test body\nmore of body",
		},
		{
			Name:          "Multiline Title",
			Message:       "test commit\nand more body",
			ExpectedTitle: "\n",
			ExpectedBody:  "test commit\nand more body",
		},
		{
			Name:          "Title includes issue number",
			Message:       "test commit #100", // # is the commentChar
			ExpectedTitle: "test commit #100",
			ExpectedBody:  "",
		},
		{
			Name:          "commented lines",
			Message:       "# this is a comment\nThe title\n\nThe Body\n# another coment", // # is the commentChar
			ExpectedTitle: "The title",
			ExpectedBody:  "The Body",
		},
		{
			Name:          "escaped commented lines",
			Message:       "# this is a comment\nThe title\n\nThe Body\n\\# markdown title", // # is the commentChar
			ExpectedTitle: "The title",
			ExpectedBody:  "The Body\n# markdown title",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			title, body, err := ParseTitleBody(test.Message)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, title, test.ExpectedTitle)
			assert.Equal(t, body, test.ExpectedBody)
		})
	}
}

func TestEditor(t *testing.T) {
	filePath := filepath.Join(os.TempDir(), "labEditorTest")
	if _, err := os.Stat(filePath); err == os.ErrExist {
		os.Remove(filePath)
	}

	var path string
	t.Run("editor()", func(t *testing.T) {
		var err error
		path, err = editor()
		if err != nil {
			t.Fatal(err)
		}

		require.NotEmpty(t, editor)
	})
	t.Run("Open Editor", func(t *testing.T) {
		cmd, err := editorCMD(path, filePath)
		if err != nil {
			t.Fatal(err)
		}
		err = cmd.Start()
		if err != nil {
			t.Fatal(err)
		}
		err = cmd.Process.Kill()
		if err != nil {
			t.Fatal(err)
		}
	})
}
