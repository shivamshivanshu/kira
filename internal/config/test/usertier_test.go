package config_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func writeRepoConfig(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, storage.ConfigRelPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func userTierEnv(t *testing.T) (dir string, env func(string) string) {
	t.Helper()
	home := t.TempDir()
	dir = filepath.Join(home, ".config", "kira")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir, func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}
}

func writeUserFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const minimalRepo = "version: 2\nproject:\n  key: KIRA\n"

func loadWith(t *testing.T, repoBody string, env func(string) string) (*datamodel.Config, string) {
	t.Helper()
	root := writeRepoConfig(t, repoBody)
	var warn bytes.Buffer
	cfg, err := config.LoadWithUser(root, env, &warn)
	if err != nil {
		t.Fatalf("LoadWithUser: %v", err)
	}
	return cfg, warn.String()
}

func TestUserConfigDirResolution(t *testing.T) {
	t.Parallel()
	xdg := func(k string) string {
		switch k {
		case "XDG_CONFIG_HOME":
			return "/xdg"
		case "HOME":
			return "/home/u"
		}
		return ""
	}
	if dir, ok := config.UserConfigDir(xdg); !ok || dir != filepath.Join("/xdg", "kira") {
		t.Fatalf("XDG_CONFIG_HOME should win: got %q ok=%v", dir, ok)
	}

	homeOnly := func(k string) string {
		if k == "HOME" {
			return "/home/u"
		}
		return ""
	}
	if dir, ok := config.UserConfigDir(homeOnly); !ok || dir != filepath.Join("/home/u", ".config", "kira") {
		t.Fatalf("HOME fallback wrong: got %q ok=%v", dir, ok)
	}

	if _, ok := config.UserConfigDir(func(string) string { return "" }); ok {
		t.Fatal("no HOME and no XDG_CONFIG_HOME must yield no user tier")
	}
}

func TestSandboxUnsetHomeYieldsNoTier(t *testing.T) {
	t.Parallel()
	cfg, warn := loadWith(t, minimalRepo, func(string) string { return "" })
	if warn != "" {
		t.Fatalf("unset home must not warn, got %q", warn)
	}
	if cfg.UI.Icons != datamodel.IconAuto {
		t.Fatal("unset home must leave builtin defaults untouched")
	}
}

func TestUIUserOverridesBuiltinRepoOverridesUser(t *testing.T) {
	t.Parallel()
	dir, env := userTierEnv(t)
	writeUserFile(t, dir, "config.yaml", "ui:\n  icons: text\n")

	cfg, warn := loadWith(t, minimalRepo, env)
	if warn != "" {
		t.Fatalf("clean user ui must not warn, got %q", warn)
	}
	if cfg.UI.Icons != datamodel.IconText {
		t.Fatalf("user ui.icons should override builtin: got %q", cfg.UI.Icons)
	}
	if cfg.UI.Background != datamodel.BackgroundDark {
		t.Fatalf("unset user subkey must keep builtin default: got %q", cfg.UI.Background)
	}

	cfg, _ = loadWith(t, minimalRepo+"ui:\n  icons: nerd\n", env)
	if cfg.UI.Icons != datamodel.IconNerd {
		t.Fatalf("repo ui.icons must beat user: got %q", cfg.UI.Icons)
	}
}

func TestRepoAuthoritativeKeyIgnoredWithWarning(t *testing.T) {
	t.Parallel()
	dir, env := userTierEnv(t)
	writeUserFile(t, dir, "config.yaml", "project:\n  key: HIJACK\npriorities:\n  - X\n")
	cfg, warn := loadWith(t, minimalRepo, env)

	if cfg.Project.Key != "KIRA" {
		t.Fatalf("repo must stay authoritative for project.key: got %q", cfg.Project.Key)
	}
	for _, key := range []string{"project", "priorities"} {
		if !strings.Contains(warn, "key \""+key+"\" is repo-authoritative; ignored") {
			t.Fatalf("missing ignored-key warning for %q in %q", key, warn)
		}
	}
}

func TestMalformedUserFileWarnsNotFatal(t *testing.T) {
	t.Parallel()
	dir, env := userTierEnv(t)
	writeUserFile(t, dir, "config.yaml", "ui: [unterminated\n")
	cfg, warn := loadWith(t, minimalRepo, env)
	if warn == "" {
		t.Fatal("malformed user file must warn")
	}
	if cfg.UI.Icons != datamodel.IconAuto {
		t.Fatalf("malformed user file must fall back to builtin: got %q", cfg.UI.Icons)
	}
}

func TestEmptyUserFileSilentlyIgnored(t *testing.T) {
	t.Parallel()
	dir, env := userTierEnv(t)
	writeUserFile(t, dir, "config.yaml", "")
	cfg, warn := loadWith(t, minimalRepo, env)
	if warn != "" {
		t.Fatalf("empty user file must not warn, got %q", warn)
	}
	if cfg.UI.Icons != datamodel.IconAuto {
		t.Fatal("empty user file must leave defaults")
	}
}

func TestInvalidUserUIIgnored(t *testing.T) {
	t.Parallel()
	dir, env := userTierEnv(t)
	writeUserFile(t, dir, "config.yaml", "ui:\n  icons: bogus\n")
	cfg, warn := loadWith(t, minimalRepo, env)
	if !strings.Contains(warn, "ui.icons") {
		t.Fatalf("invalid user ui must warn about ui.icons, got %q", warn)
	}
	if cfg.UI.Icons != datamodel.IconAuto {
		t.Fatalf("invalid user ui must be ignored: got %q", cfg.UI.Icons)
	}
}

func TestUserCommitSubjectApplies(t *testing.T) {
	t.Parallel()
	dir, env := userTierEnv(t)
	writeUserFile(t, dir, "config.yaml", "commit:\n  subject: 'tickets: {numbers}'\n")

	cfg, warnings := loadWith(t, minimalRepo, env)
	if cfg.UserCommitSubject != "tickets: {numbers}" {
		t.Errorf("UserCommitSubject = %q", cfg.UserCommitSubject)
	}
	if strings.Contains(warnings, "commit") {
		t.Errorf("commit key must be accepted in the user tier, got warning: %q", warnings)
	}
}

func TestUserCommitSubjectRepoCannotOverride(t *testing.T) {
	t.Parallel()
	dir, env := userTierEnv(t)
	writeUserFile(t, dir, "config.yaml", "commit:\n  subject: 'mine {count}'\n")

	repo := minimalRepo + "commit:\n  mode: manual\n"
	cfg, _ := loadWith(t, repo, env)
	if cfg.UserCommitSubject != "mine {count}" {
		t.Errorf("repo tier clobbered user commit subject: %q", cfg.UserCommitSubject)
	}
	if cfg.Commit.Mode != datamodel.CommitManual {
		t.Errorf("repo commit.mode lost: %q", cfg.Commit.Mode)
	}
}

func TestUserCommitSubjectMultilineIgnored(t *testing.T) {
	t.Parallel()
	dir, env := userTierEnv(t)
	writeUserFile(t, dir, "config.yaml", "commit:\n  subject: \"a\\nb\"\n")

	cfg, warnings := loadWith(t, minimalRepo, env)
	if cfg.UserCommitSubject != "" {
		t.Errorf("multi-line subject must be ignored, got %q", cfg.UserCommitSubject)
	}
	if !strings.Contains(warnings, "single line") {
		t.Errorf("expected single-line warning, got %q", warnings)
	}
}
