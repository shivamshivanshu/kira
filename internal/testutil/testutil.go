// Package testutil holds shared test helpers for spinning up hermetic git repos.
package testutil

import (
	"os"
	"testing"

	"github.com/shivamshivanshu/kira/internal/gitx"
)

func GitInit(dir string) error {
	repo := gitx.Repo{Dir: dir}
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "tester"},
	} {
		if _, err := repo.Output(args...); err != nil {
			return err
		}
	}
	return nil
}

func InitGitRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("EDITOR", "true")
	root := t.TempDir()
	if err := GitInit(root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	return root
}
