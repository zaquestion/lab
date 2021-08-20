package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: tests for other functions, like createNote, are part of the
// issue_note test suite.

func Test_noteMsg(t *testing.T) {
	tests := []struct {
		Name         string
		Msgs         []string
		ExpectedBody string
	}{
		{
			Name:         "Using messages",
			Msgs:         []string{"note paragraph 1", "note paragraph 2"},
			ExpectedBody: "note paragraph 1\n\nnote paragraph 2",
		},
		{
			Name:         "From Editor",
			Msgs:         nil,
			ExpectedBody: "", // this is not a great test
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			body, err := noteMsg(test.Msgs, false, 1, "OPEN", "", "\n")
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, test.ExpectedBody, body)
		})
	}
}

func Test_noteText(t *testing.T) {
	t.Parallel()
	tmpl := noteGetTemplate(true, "")
	text, err := noteText(1701, "OPEN", "", "\n", tmpl)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, `

# This comment is being applied to OPEN Merge Request 1701.
# Comment lines beginning with '#' are discarded.`, text)
}
