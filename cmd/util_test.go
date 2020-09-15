package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_textToMarkdown(t *testing.T) {
	basestring := "This string should have two spaces at the end."
	teststring := basestring + "\n"
	newteststring := textToMarkdown(teststring)
	assert.Equal(t, basestring+"  \n", newteststring)
}
