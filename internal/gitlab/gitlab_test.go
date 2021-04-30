package gitlab

import (
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/otiai10/copy"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "github.com/xanzy/go-gitlab"
)

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	repo := copyTestRepo()
	err := os.Chdir(repo)
	if err != nil {
		log.Fatal(err)
	}

	viper.SetConfigName("lab")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	err = viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
	host := viper.GetString("core.host")
	token := viper.GetString("core.token")

	lab, _ := gitlab.NewClient(token, gitlab.WithBaseURL(host+"/api/v4"))
	u, _, err := lab.Users.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}

	Init(host, u.Username, token, false)

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

func TestGetCommit(t *testing.T) {
	tests := []struct {
		desc     string
		ref      string
		ok       bool
		expectID string
	}{
		{
			"not pushed",
			"not_a_branch",
			false,
			"",
		},
		{
			"pushed branch",
			"mrtest", // branch name
			true,
			"54fd49a2ac60aeeef5ddc75efecd49f85f7ba9b0",
		},
		{
			"pushed branch, neeeds encoding",
			"needs/encode", // branch name
			true,
			"381f2b123dd404e8046ea42d5785061aa3b6674b",
		},
		{
			"pushed sha",
			"700e056463504690c11d63727bf25a380f303be9",
			true,
			"700e056463504690c11d63727bf25a380f303be9",
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			test := test
			t.Parallel()
			b, err := GetCommit(4181224, test.ref)
			if test.ok {
				require.NoError(t, err)
				require.Equal(t, test.expectID, b.ID)
			} else {
				require.Error(t, err)
				require.Nil(t, b)
			}
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
	if err := copy.Copy(dst+"/test.git", dst+"/.git"); err != nil {
		log.Fatal(err)
	}
	return dst
}
