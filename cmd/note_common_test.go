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
			body, err := noteMsg(test.Msgs, false, "\n")
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, test.ExpectedBody, body)
		})
	}
}

func Test_noteText(t *testing.T) {
	t.Parallel()
	text, err := noteText("\n")
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, `

# Write a message for this note. Commented lines are discarded.`, text)

}
