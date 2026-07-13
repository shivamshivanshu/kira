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

const minimalRepo = "version: 1\nproject:\n  key: KIRA\n"

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
	cfg, warn := loadWith(t, minimalRepo, func(string) string { return "" })
	if warn != "" {
		t.Fatalf("unset home must not warn, got %q", warn)
	}
	if cfg.UI.Icons != datamodel.IconAuto || len(cfg.UserAutomation) != 0 {
		t.Fatal("unset home must leave builtin defaults untouched")
	}
}

func TestUIUserOverridesBuiltinRepoOverridesUser(t *testing.T) {
	dir, env := userTierEnv(t)
	writeUserFile(t, dir, "config.yaml", "ui:\n  icons: text\n")

	cfg, warn := loadWith(t, minimalRepo, env)
	if warn != "" {
		t.Fatalf("clean user ui must not warn, got %q", warn)
	}
	if cfg.UI.Icons != datamodel.IconText {
		t.Fatalf("user ui.icons should override builtin: got %q", cfg.UI.Icons)
	}
	if cfg.UI.Background != datamodel.BackgroundAuto {
		t.Fatalf("unset user subkey must keep builtin default: got %q", cfg.UI.Background)
	}

	cfg, _ = loadWith(t, minimalRepo+"ui:\n  icons: nerd\n", env)
	if cfg.UI.Icons != datamodel.IconNerd {
		t.Fatalf("repo ui.icons must beat user: got %q", cfg.UI.Icons)
	}
}

func TestRepoAuthoritativeKeyIgnoredWithWarning(t *testing.T) {
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

func TestMisplacedAutomationKeyInConfigWarns(t *testing.T) {
	dir, env := userTierEnv(t)
	writeUserFile(t, dir, "config.yaml", "automation:\n  - on: item.created\n    run: 'true'\n")
	cfg, warn := loadWith(t, minimalRepo, env)

	if !strings.Contains(warn, "personal hooks belong in hooks.yaml; ignored") {
		t.Fatalf("misplaced automation key must warn, got %q", warn)
	}
	if len(cfg.UserAutomation) != 0 {
		t.Fatalf("automation in config.yaml must not load: %+v", cfg.UserAutomation)
	}
}

func TestMalformedUserFileWarnsNotFatal(t *testing.T) {
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

func TestUserHooksFromHooksYAML(t *testing.T) {
	dir, env := userTierEnv(t)
	writeUserFile(t, dir, "hooks.yaml", "- name: mine\n  on: item.created\n  run: touch marker\n")
	cfg, warn := loadWith(t, minimalRepo, env)
	if warn != "" {
		t.Fatalf("valid user hooks must not warn, got %q", warn)
	}
	if len(cfg.Automation) != 0 {
		t.Fatal("user hooks must not leak into repo Automation")
	}
	if len(cfg.UserAutomation) != 1 || cfg.UserAutomation[0].Name != "mine" {
		t.Fatalf("user hook not loaded: %+v", cfg.UserAutomation)
	}
}

func TestUserHooksFromHooksJSON(t *testing.T) {
	dir, env := userTierEnv(t)
	writeUserFile(t, dir, "hooks.json", `[{"name":"j","on":"item.created","run":"touch marker"}]`)
	cfg, warn := loadWith(t, minimalRepo, env)
	if warn != "" {
		t.Fatalf("valid json hooks must not warn, got %q", warn)
	}
	if len(cfg.UserAutomation) != 1 || cfg.UserAutomation[0].Name != "j" {
		t.Fatalf("json hook not loaded: %+v", cfg.UserAutomation)
	}
}

func TestBothHookExtensionsYAMLWins(t *testing.T) {
	dir, env := userTierEnv(t)
	writeUserFile(t, dir, "hooks.yaml", "- name: fromyaml\n  on: item.created\n  run: 'true'\n")
	writeUserFile(t, dir, "hooks.json", `[{"name":"fromjson","on":"item.created","run":"true"}]`)
	cfg, warn := loadWith(t, minimalRepo, env)

	if !strings.Contains(warn, "hooks.json") || !strings.Contains(warn, "shadowed by hooks.yaml; ignored") {
		t.Fatalf("both extensions must warn that json is ignored, got %q", warn)
	}
	if len(cfg.UserAutomation) != 1 || cfg.UserAutomation[0].Name != "fromyaml" {
		t.Fatalf("hooks.yaml must win over hooks.json: %+v", cfg.UserAutomation)
	}
}

func TestInvalidUserHooksIgnored(t *testing.T) {
	dir, env := userTierEnv(t)
	writeUserFile(t, dir, "hooks.yaml", "- name: bad\n  on: not.an.event\n  run: 'true'\n")
	cfg, warn := loadWith(t, minimalRepo, env)
	if warn == "" {
		t.Fatal("invalid user hook must warn")
	}
	if len(cfg.UserAutomation) != 0 {
		t.Fatalf("invalid user hook must be dropped: %+v", cfg.UserAutomation)
	}
}
