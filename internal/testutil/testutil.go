// Package testutil holds shared test helpers for spinning up hermetic git repos.
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/gitx"
)

// GitInit initializes dir as a git repo with a hermetic test identity.
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

func init() {
	_ = os.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	_ = os.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	_ = os.Setenv("VISUAL", "")
	_ = os.Setenv("EDITOR", "true")

	neutralUserConfigDir := filepath.Join(os.TempDir(), fmt.Sprintf("kira-testutil-xdg-%d", os.Getpid()))
	_ = os.Setenv("XDG_CONFIG_HOME", neutralUserConfigDir)
}

// InitGitRepo creates a temp dir, initializes it as a git repo, and returns
// its path. It fails the test on error.
func InitGitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := GitInit(root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	return root
}
