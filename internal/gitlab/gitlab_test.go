package gitlab

import (
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zaquestion/lab/internal/copy"
)

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	repo := copyTestRepo()
	err := os.Chdir(repo)
	if err != nil {
		log.Fatal(err)
	}

	viper.SetConfigName("lab")
	viper.SetConfigType("hcl")
	viper.AddConfigPath(".")
	err = viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
	c := viper.AllSettings()["core"]
	config := c.([]map[string]interface{})[0]

	Init(
		config["host"].(string),
		config["token"].(string))

	code := m.Run()

	if err := os.Chdir("../"); err != nil {
		log.Fatalf("Error chdir to ../: %s", err)
	}
	if err := os.RemoveAll(repo); err != nil {
		log.Fatalf("Error removing %s: %s", repo, err)
	}
	os.Exit(code)
}

func TestGetProject(t *testing.T) {
	project, err := GetProject("lab-testing/test")
	require.NoError(t, err)
	assert.Equal(t, 5694926, project.ID, "Expected 'lab-testing/test' to be project 5694926")
}

func TestUser(t *testing.T) {
	// Should get set by Init() after TestMain()
	require.Equal(t, "lab-testing", User())
}

func TestLoadGitLabTmplMR(t *testing.T) {
	mrTmpl := LoadGitLabTmpl(TmplMR)
	require.Equal(t, "I am the default merge request template for lab", mrTmpl)
}

func TestLoadGitLabTmplIssue(t *testing.T) {
	issueTmpl := LoadGitLabTmpl(TmplIssue)
	require.Equal(t, "This is the default issue template for lab", issueTmpl)
}

func TestLint(t *testing.T) {
	tests := []struct {
		desc     string
		content  string
		expected bool
	}{
		{
			"Valid",
			`build1:
  stage: build
  script:
    - echo "Do your build here"`,
			true,
		},
		{
			"Invalid",
			`build1:
    - echo "Do your build here"`,
			false,
		},
		{
			"Empty",
			``,
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			test := test
			t.Parallel()
			ok, _ := Lint(test.content)
			require.Equal(t, test.expected, ok)
		})
	}
}

func TestBranchPushed(t *testing.T) {
	tests := []struct {
		desc     string
		branch   string
		expected bool
	}{
		{
			"alpha is pushed",
			"mrtest",
			true,
		},
		{
			"needs encoding is pushed",
			"needs/encode",
			true,
		},
		{
			"alpha not pushed",
			"not_a_branch",
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			test := test
			t.Parallel()
			ok := BranchPushed(4181224, test.branch)
			require.Equal(t, test.expected, ok)
		})
	}
}

// copyTestRepo creates a copy of the testdata directory (contains a Git repo) in
// the project root with a random dir name. It returns the absolute path of the
// new testdata dir.
// Note: testdata-* must be in the .gitignore or the copies will create write
// errors as Git attempts to add the Git repo to the the project repo's index.
func copyTestRepo() string {
	dst, err := filepath.Abs(os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata-" + strconv.Itoa(int(rand.Uint64()))))
	if err != nil {
		log.Fatal(err)
	}
	src, err := filepath.Abs(os.ExpandEnv("$GOPATH/src/github.com/zaquestion/lab/testdata"))
	if err != nil {
		log.Fatal(err)
	}
	if err := copy.Copy(src, dst); err != nil {
		log.Fatal(err)
	}
	// Move the test.git dir into the expected path at .git
	if err := os.Rename(dst+"/test.git", dst+"/.git"); err != nil {
		log.Fatal(err)
	}
	return dst
}
