package cmd

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ciStatus(t *testing.T) {
	t.Parallel()
	repo := copyTestRepo(t)
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = repo
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	cmd = exec.Command("git", "checkout", "-b", "ci_test_pipeline")
	cmd.Dir = repo
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	cmd = exec.Command("git", "branch", "-m", "local/ci_test_pipeline")
	cmd.Dir = repo
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	cmd = exec.Command(labBinaryPath, "ci", "status")
	cmd.Dir = repo

	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
	out := string(b)
	assert.Contains(t, out, `Stage:  Name                           - Status
build:  build1                         - success
build:  build2                         - success
build:  build2:fails                   - failed
test:   test1                          - success
test:   test2                          - success
test:   test2:really_a_long_name_for   - success
test:   test2:no_suffix:test           - success
test:   test3                          - success
deploy: deploy1                        - success
deploy: deploy2                        - manual
deploy: deploy3:no_sufix:deploy        - success
deploy: deploy4                        - success
deploy: deploy5:really_a_long_name_for - success
deploy: deploy5                        - success
deploy: deploy6                        - success
deploy: deploy7                        - success
deploy: deploy8                        - success
deploy: deploy9                        - success
deploy: deploy10                       - success`)

	assert.Contains(t, out, "Pipeline Status: success")
}
