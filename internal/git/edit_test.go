package git

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func Test_parseTitleBody(t *testing.T) {
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
			Message:       "test commit\nand more title",
			ExpectedTitle: "test commit and more title",
			ExpectedBody:  "",
		},
		{
			Name:          "Title includes issue number",
			Message:       "test commit #100", // # is the commentChar
			ExpectedTitle: "test commit #100",
			ExpectedBody:  "",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			title, body, err := parseTitleBody(test.Message)
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
	t.Run("editorPath()", func(t *testing.T) {
		var err error
		path, err = editorPath()
		if err != nil {
			t.Fatal(err)
		}

		require.NotEmpty(t, editorPath)
	})
	t.Run("Open Editor", func(t *testing.T) {
		cmd := editorCMD(path, filePath)
		err := cmd.Start()
		if err != nil {
			t.Fatal(err)
		}
		err = cmd.Process.Kill()
		if err != nil {
			t.Fatal(err)
		}
	})
}
