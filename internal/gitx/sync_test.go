package gitx

import (
	"os"
	"path/filepath"
	"testing"
)

func commitFile(t *testing.T, dir, content, msg string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "f.txt")
	gitRun(t, dir, "commit", "-m", msg)
}

func TestRebaseContinueOverridesUserGitEditor(t *testing.T) {
	dir := t.TempDir()
	gitRun(t, dir, "init", "-b", "main")
	gitRun(t, dir, "config", "user.name", "t")
	gitRun(t, dir, "config", "user.email", "t@e.c")
	gitRun(t, dir, "config", "commit.gpgsign", "false")
	commitFile(t, dir, "base\n", "base")
	gitRun(t, dir, "checkout", "-b", "feature")
	commitFile(t, dir, "feature\n", "feature")
	gitRun(t, dir, "checkout", "main")
	commitFile(t, dir, "main\n", "main")
	gitRun(t, dir, "checkout", "feature")

	if _, err := gitTry(dir, "rebase", "main"); err == nil {
		t.Fatal("rebase main: want conflict, got success")
	}
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("resolved\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "f.txt")

	t.Setenv("GIT_EDITOR", "false")
	repo := Repo{Dir: dir}
	if err := repo.RebaseContinue(); err != nil {
		t.Fatalf("RebaseContinue with hostile GIT_EDITOR: %v", err)
	}
	if repo.RebaseInProgress() {
		t.Fatal("rebase still in progress after RebaseContinue")
	}
}
