package copy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	os.MkdirAll("testdata.copy", os.ModePerm)
	code := m.Run()
	os.RemoveAll("testdata.copy")
	os.Exit(code)
}

func TestCopy(t *testing.T) {

	require.NoError(t, Copy("./testdata/case00", "./testdata.copy/case00"))
	info, err := os.Stat("./testdata.copy/case00/README.md")
	require.NoError(t, err)
	assert.False(t, info.IsDir())

	assert.Error(
		t,
		Copy("NOT/EXISTING/SOURCE/PATH", "anywhere"),
		"Expected error when src doesn't exist")
	assert.NoError(
		t,
		Copy("testdata/case01/README.md", "testdata.copy/case01/README.md"),
		"No error when src is just a file")

	dest := "foobar"
	for i := 0; i < 8; i++ {
		dest = dest + dest
	}
	err = Copy("testdata/case00", filepath.Join("testdata/case00", dest))
	assert.Error(t, err)
	assert.IsType(t, &os.PathError{}, err, "Expected error when filename is too long")

	err = Copy("testdata/case02", "testdata.copy/case00/README.md")
	assert.Error(t, err)
	assert.IsType(
		t,
		&os.PathError{},
		err,
		"Expect error when creating a directory on existing filename")

	assert.Error(
		t,
		Copy("testdata/case04/README.md", "testdata/case04"),
		"Expected error when copying file to an existing dir")
	assert.Error(
		t,
		Copy("testdata/case04/README.md", "testdata/case04/README.md/foobar"),
		"Expected error when copying file to an existing file")
}
