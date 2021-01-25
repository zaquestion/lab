package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "github.com/xanzy/go-gitlab"
)

func Test_editGetTitleAndDescription(t *testing.T) {
	tests := []struct {
		Name                string
		Issue               *gitlab.Issue
		Args                []string
		ExpectedTitle       string
		ExpectedDescription string
	}{
		{
			Name: "Using messages",
			Issue: &gitlab.Issue{
				Title:       "old title",
				Description: "old body",
			},
			Args:                []string{"new title", "new body 1", "new body 2"},
			ExpectedTitle:       "new title",
			ExpectedDescription: "new body 1\n\nnew body 2",
		},
		{
			Name: "Using a single message",
			Issue: &gitlab.Issue{
				Title:       "old title",
				Description: "old body",
			},
			Args:                []string{"new title"},
			ExpectedTitle:       "new title",
			ExpectedDescription: "old body",
		},
		{
			Name: "From Editor",
			Issue: &gitlab.Issue{
				Title:       "old title",
				Description: "old body",
			},
			Args:                nil,
			ExpectedTitle:       "old title",
			ExpectedDescription: "old body",
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			title, body, err := editGetTitleDescription(test.Issue.Title, test.Issue.Description, test.Args, len(test.Args))
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
