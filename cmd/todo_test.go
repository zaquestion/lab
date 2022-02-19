package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// a simple helper to read the todo list
func readTodoList(t *testing.T) string {
	cmd := exec.Command(labBinaryPath, "todo", "list", "lab-testing")
	l, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(l))
		t.Fatal(err)
	}
	return string(l)
}

func Test_todoMergeRequestTest(t *testing.T) {
	repo := copyTestRepo(t)

	todoListOrig := readTodoList(t)

	// create a Merge Request
	git := exec.Command("git", "checkout", "-b", "local/mrtest", "origin/mrtest")
	git.Dir = repo
	b, err := git.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}

	cmd := exec.Command(labBinaryPath, "mr", "create", "lab-testing", "master", "-m", "mr TODO title test", "-m", "mr description")
	cmd.Dir = repo

	b, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Error creating mr: %s (%s)", string(b), err)
	}
	out := string(b)
	require.Contains(t, out, "https://gitlab.com/lab-testing/test/-/merge_requests")

	i := strings.Index(out, "/diffs\n")
	mrID := strings.TrimPrefix(out[:i], "https://gitlab.com/lab-testing/test/-/merge_requests/")

	// Add the Merge request to the Todo list
	cmd = exec.Command(labBinaryPath, "todo", "mr", "lab-testing", mrID)
	cmd.Dir = repo
	a, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(a))
		t.Fatal(err)
	}
	todoID := strings.Split(string(a), " ")[0]

	todoListAfterAdd := readTodoList(t)

	// Remove the Merge Request from the Todo list
	cmd = exec.Command(labBinaryPath, "todo", "done", todoID)
	cmd.Dir = repo
	a, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(a))
		t.Fatal(err)
	}
	doneMsg := string(a)

	todoListAfterDone := readTodoList(t)

	cmd = exec.Command(labBinaryPath, "mr", "close", "lab-testing", mrID)
	cmd.Dir = repo

	a, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(a))
		t.Fatal(err)
	}

	// At the beginning of the test, the Todo item must not be on the Todo list
	require.NotContains(t, todoListOrig, todoID)
	// The Todo item must be on the Todo list after it is added
	require.Contains(t, todoListAfterAdd, todoID)
	// The Todo item must not be on the Todo list after the Todo is marked done
	require.NotContains(t, todoListAfterDone, todoID)
	// The 'done' message must contain the Todo item
	require.Contains(t, doneMsg, todoID)
	// The 'done' message must indicate that the Todo item was marked 'Done'
	require.Contains(t, doneMsg, "marked as Done")
}

func Test_todoIssueTest(t *testing.T) {
	repo := copyTestRepo(t)

	todoListOrig := readTodoList(t)

	cmd := exec.Command(labBinaryPath, "issue", "create", "lab-testing",
		"-m", "issue title")
	cmd.Dir = repo

	a, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Error creating issue: %s (%s)", string(a), err)
	}
	out := string(a)
	require.Contains(t, out, "https://gitlab.com/lab-testing/test/-/issues")

	i := strings.Index(out, "\n")
	issueID := strings.TrimPrefix(out[:i], "https://gitlab.com/lab-testing/test/-/issues/")

	// add it to todolist
	cmd = exec.Command(labBinaryPath, "todo", "issue", "lab-testing", issueID)
	cmd.Dir = repo
	a, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(a))
		t.Fatal(err)
	}
	todoID := strings.Split(string(a), " ")[0]

	todoListAfterAdd := readTodoList(t)

	// remove it from todolist
	cmd = exec.Command(labBinaryPath, "todo", "done", todoID)
	cmd.Dir = repo
	a, err = cmd.CombinedOutput()
	if err != nil {
		t.Log(string(a))
		t.Fatal(err)
	}
	doneMsg := string(a)

	todoListAfterDone := readTodoList(t)

	// At the beginning of the test, the Todo item must not be on the Todo list
	require.NotContains(t, todoListOrig, todoID)
	// The Todo item must be on the Todo list after it is added
	require.Contains(t, todoListAfterAdd, todoID)
	// The Todo item must not be on the Todo list after the Todo is marked done
	require.NotContains(t, todoListAfterDone, todoID)
	// The 'done' message must contain the Todo item
	require.Contains(t, doneMsg, todoID)
	// The 'done' message must indicate that the Todo item was marked 'Done'
	require.Contains(t, doneMsg, "marked as Done")
}
