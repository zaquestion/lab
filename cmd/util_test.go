package cmd

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func Test_textToMarkdown(t *testing.T) {
	basestring := "This string should have two spaces at the end."
	teststring := basestring + "\n"
	newteststring := textToMarkdown(teststring)
	assert.Equal(t, basestring+"  \n", newteststring)
}

func Test_getCurrentBranchMR(t *testing.T) {
	repo := copyTestRepo(t)

	// make sure the branch does not exist
	cmd := exec.Command("git", "branch", "-D", "mrtest")
	cmd.Dir = repo
	cmd.CombinedOutput()

	cmd = exec.Command(labBinaryPath, "mr", "checkout", "1")
	cmd.Dir = repo
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	curDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	err = os.Chdir(repo)
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	mrNum := getCurrentBranchMR("zaquestion/test")
	err = os.Chdir(curDir)
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	assert.Equal(t, 1, mrNum)
}

func Test_parseArgsStringAndID(t *testing.T) {
	tests := []struct {
		Name           string
		Args           []string
		ExpectedString string
		ExpectedInt    int64
		ExpectedErr    string
	}{
		{
			Name:           "No Args",
			Args:           nil,
			ExpectedString: "",
			ExpectedInt:    0,
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg remote",
			Args:           []string{"origin"},
			ExpectedString: "origin",
			ExpectedInt:    0,
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg non remote",
			Args:           []string{"foo"},
			ExpectedString: "foo",
			ExpectedInt:    0,
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg page",
			Args:           []string{"100"},
			ExpectedString: "",
			ExpectedInt:    100,
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg invalid page",
			Args:           []string{"asdf100"},
			ExpectedString: "asdf100",
			ExpectedInt:    0,
			ExpectedErr:    "",
		},
		{
			Name:           "2 arg str page",
			Args:           []string{"origin", "100"},
			ExpectedString: "origin",
			ExpectedInt:    100,
			ExpectedErr:    "",
		},
		{
			Name:           "2 arg valid str valid page",
			Args:           []string{"foo", "100"},
			ExpectedString: "foo",
			ExpectedInt:    100,
			ExpectedErr:    "",
		},
		{
			Name:           "2 arg valid str invalid page",
			Args:           []string{"foo", "asdf100"},
			ExpectedString: "foo",
			ExpectedInt:    0,
			ExpectedErr:    "strconv.ParseInt: parsing \"asdf100\": invalid syntax",
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			s, i, err := parseArgsStringAndID(test.Args)
			if err != nil {
				assert.EqualError(t, err, test.ExpectedErr)
			}
			assert.Equal(t, test.ExpectedString, s)
			assert.Equal(t, test.ExpectedInt, i)
		})
	}
}

func Test_parseArgsRemoteAndID(t *testing.T) {
	tests := []struct {
		Name           string
		Args           []string
		ExpectedString string
		ExpectedInt    int64
		ExpectedErr    string
	}{
		{
			Name:           "No Args",
			Args:           nil,
			ExpectedString: "zaquestion/test",
			ExpectedInt:    0,
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg remote",
			Args:           []string{"lab-testing"},
			ExpectedString: "lab-testing/test",
			ExpectedInt:    0,
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg non remote",
			Args:           []string{"foo"},
			ExpectedString: "",
			ExpectedInt:    0,
			ExpectedErr:    "foo is not a valid remote or number",
		},
		{
			Name:           "1 arg page",
			Args:           []string{"100"},
			ExpectedString: "zaquestion/test",
			ExpectedInt:    100,
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg invalid page",
			Args:           []string{"asdf100"},
			ExpectedString: "",
			ExpectedInt:    0,
			ExpectedErr:    "asdf100 is not a valid remote or number",
		},
		{
			Name:           "2 arg remote page",
			Args:           []string{"origin", "100"},
			ExpectedString: "zaquestion/test",
			ExpectedInt:    100,
			ExpectedErr:    "",
		},
		{
			Name:           "2 arg invalid remote valid page",
			Args:           []string{"foo", "100"},
			ExpectedString: "",
			ExpectedInt:    0,
			ExpectedErr:    "foo is not a valid remote",
		},
		{
			Name:           "2 arg invalid remote invalid page",
			Args:           []string{"foo", "asdf100"},
			ExpectedString: "",
			ExpectedInt:    0,
			ExpectedErr:    "strconv.ParseInt: parsing \"asdf100\": invalid syntax",
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			s, i, err := parseArgsRemoteAndID(test.Args)
			if err != nil {
				assert.EqualError(t, err, test.ExpectedErr)
			}
			assert.Equal(t, test.ExpectedString, s)
			assert.Equal(t, test.ExpectedInt, i)
		})
	}
}

func Test_parseArgsRemoteAndProject(t *testing.T) {
	tests := []struct {
		Name           string
		Args           []string
		ExpectedRemote string
		ExpectedString string
		ExpectedErr    string
	}{
		{
			Name:           "No Args",
			Args:           nil,
			ExpectedRemote: "zaquestion/test",
			ExpectedString: "",
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg remote",
			Args:           []string{"lab-testing"},
			ExpectedRemote: "lab-testing/test",
			ExpectedString: "",
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg non remote",
			Args:           []string{"foo123"},
			ExpectedRemote: "zaquestion/test",
			ExpectedString: "foo123",
			ExpectedErr:    "",
		},
		{
			Name:           "1 arg page",
			Args:           []string{"100"},
			ExpectedRemote: "zaquestion/test",
			ExpectedString: "100",
			ExpectedErr:    "",
		},
		{
			Name:           "2 arg remote and string",
			Args:           []string{"origin", "foo123"},
			ExpectedRemote: "zaquestion/test",
			ExpectedString: "foo123",
			ExpectedErr:    "",
		},
		{
			Name:           "2 arg invalid remote and string",
			Args:           []string{"foo", "string123"},
			ExpectedRemote: "",
			ExpectedString: "",
			ExpectedErr:    "foo is not a valid remote",
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test := test
			t.Parallel()
			r, s, err := parseArgsRemoteAndProject(test.Args)
			if err != nil {
				assert.EqualError(t, err, test.ExpectedErr)
			}
			assert.Equal(t, test.ExpectedRemote, r)
			assert.Equal(t, test.ExpectedString, s)
		})
	}
}

func Test_labURLToRepo(t *testing.T) {
	HTTPURL := "https://test"
	SSHURL := "ssh://test"
	project := gitlab.Project{
		HTTPURLToRepo: HTTPURL,
		SSHURLToRepo:  SSHURL,
	}

	urlToRepo := labURLToRepo(&project)
	assert.Equal(t, urlToRepo, SSHURL)

	useHTTP = true
	urlToRepo = labURLToRepo(&project)
	assert.Equal(t, urlToRepo, HTTPURL)
}

func Test_determineSourceRemote(t *testing.T) {
	tests := []struct {
		desc     string
		branch   string
		expected string
	}{
		{
			desc:     "branch.<name>.remote",
			branch:   "mrtest",
			expected: "lab-testing",
		},
		{
			desc:     "branch.<name>.pushRemote",
			branch:   "mrtest-pushRemote",
			expected: "lab-testing",
		},
		{
			desc:     "pushDefault without pushRemote set",
			branch:   "mrtest",
			expected: "garbageurl",
		},
		{
			desc:     "pushDefault with pushRemote set",
			branch:   "mrtest-pushRemote",
			expected: "lab-testing",
		},
	}

	// The function being tested here depends on being in the test
	// directory, where 'git config --local' can retrieve the correct
	// info from
	repo := copyTestRepo(t)
	oldWd, err := os.Getwd()
	if err != nil {
		t.Log(err)
	}
	os.Chdir(repo)

	var remoteModified bool
	for _, test := range tests {
		test := test
		if strings.Contains(test.desc, "pushDefault") && !remoteModified {
			git := exec.Command("git", "config", "--local", "remote.pushDefault", "garbageurl")
			git.Dir = repo
			b, err := git.CombinedOutput()
			if err != nil {
				t.Log(string(b))
				t.Fatal(err)
			}
			remoteModified = true
		}

		t.Run(test.desc, func(t *testing.T) {
			sourceRemote, err := determineSourceRemote(test.branch)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, test.expected, sourceRemote)
		})
	}
	// Remove the added option to avoid messing with other tests
	git := exec.Command("git", "config", "--local", "--unset", "remote.pushDefault")
	git.Dir = repo
	b, err := git.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	// And move back to the workdir we were before the test
	os.Chdir(oldWd)
}

func Test_matchTerms(t *testing.T) {
	tests := []struct {
		desc        string
		search      []string
		existent    []string
		expected    []string
		expectedErr string
	}{
		{
			desc:        "no match",
			search:      []string{"asd", "zxc"},
			existent:    []string{"dsa", "cxz"},
			expected:    []string{""},
			expectedErr: "'asd' not found",
		},
		{
			desc:        "full match",
			search:      []string{"asd", "zxc"},
			existent:    []string{"asd", "zxc"},
			expected:    []string{"asd", "zxc"},
			expectedErr: "",
		},
		{
			desc:        "substring match",
			search:      []string{"as", "zx"},
			existent:    []string{"as", "asd", "zxc"},
			expected:    []string{"as", "zxc"},
			expectedErr: "",
		},
		{
			desc:        "ambiguous terms",
			search:      []string{"as", "zx"},
			existent:    []string{"asd", "asf", "zxc", "zxv"},
			expected:    []string{""},
			expectedErr: "'as' has no exact match and is ambiguous",
		},
	}

	t.Parallel()

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			matches, err := matchTerms(test.search, test.existent)
			if test.expected[0] != "" {
				assert.Equal(t, test.expected, matches)
			} else {
				assert.Nil(t, matches)
			}

			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func Test_same(t *testing.T) {
	t.Parallel()
	assert.True(t, same([]string{}, []string{}))
	assert.True(t, same([]string{"a"}, []string{"a"}))
	assert.True(t, same([]string{"a", "b"}, []string{"a", "b"}))
	assert.True(t, same([]string{"a", "b"}, []string{"b", "a"}))
	assert.True(t, same([]string{"b", "a"}, []string{"a", "b"}))

	assert.False(t, same([]string{"a"}, []string{}))
	assert.False(t, same([]string{"a"}, []string{"c"}))
	assert.False(t, same([]string{}, []string{"c"}))
	assert.False(t, same([]string{"a", "b"}, []string{"a", "c"}))
	assert.False(t, same([]string{"a", "b"}, []string{"a"}))
	assert.False(t, same([]string{"a", "b"}, []string{"c"}))
}

func Test_union(t *testing.T) {
	t.Parallel()
	s := union([]string{"a", "b"}, []string{"c"})
	assert.Equal(t, 3, len(s))
	assert.True(t, same(s, []string{"a", "b", "c"}))
}

func Test_difference(t *testing.T) {
	t.Parallel()
	s := difference([]string{"a", "b"}, []string{"c"})
	assert.Equal(t, 2, len(s))
	assert.True(t, same(s, []string{"a", "b"}))

	s = difference([]string{"a", "b"}, []string{"a", "c"})
	assert.Equal(t, 1, len(s))
	assert.True(t, same(s, []string{"b"}))
}
