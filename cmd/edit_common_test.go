package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_editDescription(t *testing.T) {
	repo := copyTestRepo(t)

	type fakeGLObj struct {
		Title       string
		Description string
	}

	type funcArgs struct {
		Msgs     []string
		Filename string
	}

	tests := []struct {
		Name                string
		GLObj               fakeGLObj
		Args                funcArgs
		ExpectedTitle       string
		ExpectedDescription string
	}{
		{
			Name: "Using messages",
			GLObj: fakeGLObj{
				Title:       "old title",
				Description: "old body",
			},
			Args: funcArgs{
				Msgs:     []string{"new title", "new body 1", "new body 2"},
				Filename: "",
			},
			ExpectedTitle:       "new title",
			ExpectedDescription: "new body 1\n\nnew body 2",
		},
		{
			Name: "Using a single message",
			GLObj: fakeGLObj{
				Title:       "old title",
				Description: "old body",
			},
			Args: funcArgs{
				Msgs:     []string{"new title"},
				Filename: "",
			},
			ExpectedTitle:       "new title",
			ExpectedDescription: "old body",
		},
		{
			Name: "From Editor",
			GLObj: fakeGLObj{
				Title:       "old title",
				Description: "old body",
			},
			Args: funcArgs{
				Msgs:     nil,
				Filename: "",
			},
			ExpectedTitle:       "old title",
			ExpectedDescription: "old body",
		},
		{
			Name: "From file",
			GLObj: fakeGLObj{
				Title:       "old title",
				Description: "old body",
			},
			Args: funcArgs{
				Msgs:     nil,
				Filename: filepath.Join(repo, "testedit"),
			},
			ExpectedTitle:       "new title",
			ExpectedDescription: "\nnew body\n",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			title, body, err := editDescription(test.GLObj.Title,
				test.GLObj.Description, test.Args.Msgs, test.Args.Filename)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.ExpectedTitle, title)
			assert.Equal(t, test.ExpectedDescription, body)
		})
	}
}

func Test_editText(t *testing.T) {
	t.Parallel()
	text, err := editText("old title", "old body")
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, `old title

old body

# Edit the title and/or description. The first block of text
# is the title and the rest is the description.`, text)

}
