package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ciRun(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command(labBinaryPath, "ci", "run")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	require.Regexp(t, `^https://gitlab.com/lab-testing/test/pipelines/\d+`, string(b))
}

func Test_parseCIVariables(t *testing.T) {
	t.Parallel()
	tests := []struct {
		desc        string
		vars        []string
		expected    map[string]string
		expectedErr string
	}{
		{
			"happy",
			[]string{"foo=bar", "fizz=buzz"},
			map[string]string{
				"foo":  "bar",
				"fizz": "buzz",
			},
			"",
		},
		{
			"multi equals",
			[]string{"foo=bar", "fizz=buzz=baz"},
			map[string]string{
				"foo":  "bar",
				"fizz": "buzz=baz",
			},
			"",
		},
		{
			"bad vars",
			[]string{"foo=bar", "fizzbuzz"},
			nil,
			"Invalid Variable: \"fizzbuzz\", Variables must be in the format key=value",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			ciVars, err := parseCIVariables(test.vars)
			assert.Equal(t, test.expected, ciVars)
			if test.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, test.expectedErr)
			}
		})
	}
}

func Test_getCIRunOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		desc            string
		cmdFunc         func()
		args            []string
		expectedProject interface{}
		expectedBranch  string
		expectedErr     string
	}{
		{
			"noargs",
			nil,
			[]string{},
			"zaquestion/test",
			"master",
			"",
		},
		{
			"branch arg",
			nil,
			[]string{"mybranch"},
			"zaquestion/test",
			"mybranch",
			"",
		},
		{
			"fork branch arg",
			nil,
			[]string{"mrtest"},
			"lab-testing/test",
			"mrtest",
			"",
		},
		{
			"project flag",
			func() {
				ciTriggerCmd.Flags().Set("project", "zaquestion/test")
			},
			[]string{},
			4181224, // https://gitlab.com/zaquestion/test project ID
			"master",
			"",
		},
		{
			"bad project",
			func() {
				ciTriggerCmd.Flags().Set("project", "barfasdfasdf")
			},
			[]string{},
			nil, // https://gitlab.com/zaquestion/test project ID
			"",
			"gitlab project not found",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			ciTriggerCmd.Flags().Set("project", "")
			if test.cmdFunc != nil {
				test.cmdFunc()
			}

			p, b, err := getCIRunOptions(ciTriggerCmd, test.args)
			assert.Equal(t, test.expectedProject, p)
			assert.Equal(t, test.expectedBranch, b)
			if test.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, test.expectedErr)
			}
		})
	}
}
