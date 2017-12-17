package cmd

import "testing"

func Test_snippetCmd(t *testing.T) {
	var snipID string
	t.Run("create", func(t *testing.T) {
		repo := copyTestRepo(t)
		cmd := exec.Command("../lab_bin", "snippet", "create", "-g",
			"-m", "personal snippet title",
			"-m", "personal snippet description")
		cmd.Dir = repo

		rc, err := cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}

		_, err = rc.Write([]byte("personal snippet contents"))
		if err != nil {
			t.Fatal(err)
		}
		err = rc.Close()
		if err != nil {
			t.Fatal(err)
		}

		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Log(string(b))
			t.Fatal(err)
		}

		require.Contains(t, string(b), "https://gitlab.com/snippets/")
		snipID := strings.TrimPrefix("https://gitlab.com/snippets/")
	})
	t.Run("delete", func(t *testing.T) {
		repo := copyTestRepo(t)
	})
}
