package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ciTrace(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("../lab_bin", "fetch", "origin")
	cmd.Dir = repo
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	cmd = exec.Command("../lab_bin", "checkout", "origin/ci_test_pipeline")
	cmd.Dir = repo
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	cmd = exec.Command("../lab_bin", "checkout", "-b", "ci_test_pipeline")
	cmd.Dir = repo
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	tests := []struct {
		desc           string
		args           []string
		assertContains func(t *testing.T, out string)
	}{
		{
			desc: "noargs",
			args: []string{},
			assertContains: func(t *testing.T, out string) {
				assert.Contains(t, out, "Showing logs for deploy10")
				assert.Contains(t, out, "Checking out 09b519cb as ci_test_pipeline...")
				assert.Contains(t, out, "For example you might run an update here or install a build dependency")
				assert.Contains(t, out, "$ echo \"Or perhaps you might print out some debugging details\"")
				assert.Contains(t, out, "Job succeeded")
			},
		},
		{
			desc: "manual",
			args: []string{"origin", "deploy2"},
			assertContains: func(t *testing.T, out string) {
				assert.Contains(t, out, "Manual job deploy2 not started\n")
			},
		},
		{
			desc: "arg job name",
			args: []string{"origin", "deploy1"},
			assertContains: func(t *testing.T, out string) {
				assert.Contains(t, out, "Showing logs for deploy1")
				assert.Contains(t, out, "Checking out 09b519cb as ci_test_pipeline...")
				assert.Contains(t, out, "For example you might run an update here or install a build dependency")
				assert.Contains(t, out, "$ echo \"Or perhaps you might print out some debugging details\"")
				assert.Contains(t, out, "Job succeeded")
			},
		},
		{
			desc: "explicit branch:job",
			args: []string{"origin", "ci_test_pipeline:deploy1"},
			assertContains: func(t *testing.T, out string) {
				assert.Contains(t, out, "Showing logs for deploy1")
				assert.Contains(t, out, "Checking out 09b519cb as ci_test_pipeline...")
				assert.Contains(t, out, "For example you might do some cleanup here")
				assert.Contains(t, out, "Job succeeded")
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			cmd = exec.Command("../lab_bin", append([]string{"ci", "trace"}, test.args...)...)
			cmd.Dir = repo

			b, err := cmd.CombinedOutput()
			if err != nil {
				t.Log(string(b))
				t.Fatal(err)
			}
			out := string(b)
			test.assertContains(t, out)
		})
	}

}
