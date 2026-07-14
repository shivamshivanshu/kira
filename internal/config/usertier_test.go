package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/storage"
)

const minimalRepo = "version: 1\nproject:\n  key: KIRA\n"

func loadWithUser(t *testing.T, repoYAML, userYAML string) (*datamodel.Config, string) {
	t.Helper()
	root, xdg := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(root, storage.ConfigRelPath), repoYAML)
	if userYAML != "" {
		writeFile(t, filepath.Join(xdg, "kira", "config.yaml"), userYAML)
	}
	env := func(k string) string {
		if k == "XDG_CONFIG_HOME" {
			return xdg
		}
		return ""
	}
	var warn bytes.Buffer
	cfg, err := LoadWithUser(root, env, &warn)
	if err != nil {
		t.Fatalf("LoadWithUser: %v", err)
	}
	return cfg, warn.String()
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestUserWorkonHonored(t *testing.T) {
	cfg, _ := loadWithUser(t, minimalRepo, "workon:\n  worktree: true\n  worktree_dir: ../wt/{number}\n")
	if !cfg.Workon.Worktree {
		t.Error("user workon.worktree not honored")
	}
	if cfg.Workon.WorktreeDir != "../wt/{number}" {
		t.Errorf("user workon.worktree_dir = %q", cfg.Workon.WorktreeDir)
	}
}

func TestRepoWinsOverUserWorkon(t *testing.T) {
	repo := minimalRepo + "workon:\n  worktree_dir: ../repo-wins\n"
	cfg, _ := loadWithUser(t, repo, "workon:\n  worktree: true\n  worktree_dir: ../user-loses\n")
	if cfg.Workon.WorktreeDir != "../repo-wins" {
		t.Errorf("repo did not win worktree_dir: %q", cfg.Workon.WorktreeDir)
	}
	if !cfg.Workon.Worktree {
		t.Error("user worktree:true should survive where repo left it unset")
	}
}

func TestUserUnknownKeyWarns(t *testing.T) {
	_, warn := loadWithUser(t, minimalRepo, "priorities:\n  - X\n")
	if !bytes.Contains([]byte(warn), []byte(`key "priorities" is repo-authoritative`)) {
		t.Errorf("unknown user key did not warn: %q", warn)
	}
}

func TestUserEditorHonored(t *testing.T) {
	cfg, warn := loadWithUser(t, minimalRepo, "ui:\n  editor: vim -u NONE\n")
	if cfg.UI.Editor != "vim -u NONE" {
		t.Errorf("user ui.editor = %q", cfg.UI.Editor)
	}
	if warn != "" {
		t.Errorf("unexpected warnings: %q", warn)
	}
}

func TestRepoEditorIgnored(t *testing.T) {
	repo := minimalRepo + "ui:\n  editor: rm -rf /\n"
	cfg, _ := loadWithUser(t, repo, "ui:\n  editor: safe-editor\n")
	if cfg.UI.Editor != "safe-editor" {
		t.Errorf("repo ui.editor overrode the user tier: %q", cfg.UI.Editor)
	}
	cfg, _ = loadWithUser(t, repo, "")
	if cfg.UI.Editor != "" {
		t.Errorf("repo ui.editor must be ignored, got %q", cfg.UI.Editor)
	}
}

func TestUserBadColumnWarns(t *testing.T) {
	_, warn := loadWithUser(t, minimalRepo, "ui:\n  list:\n    columns: [number, bogus, title]\n")
	if !bytes.Contains([]byte(warn), []byte(`unknown column "bogus"`)) {
		t.Errorf("bad column did not warn: %q", warn)
	}
}
