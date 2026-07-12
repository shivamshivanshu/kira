package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
)

// writeStore lays down a minimal .kira/ tree (config + tickets) under a temp
// dir and returns the root. It does not init git; tests that need commits use
// the e2e harness.
func writeStore(t *testing.T, tickets map[string]string) string {
	t.Helper()
	root := t.TempDir()
	kira := filepath.Join(root, ".kira")
	if err := os.MkdirAll(filepath.Join(kira, "tickets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kira, "config.yaml"), []byte(initConfigYAML("KIRA", "kira")), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, content := range tickets {
		if err := os.WriteFile(filepath.Join(kira, "tickets", name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func sampleItem(ulid, number, title string) string {
	return "---\n" +
		"id: " + ulid + "\n" +
		"number: " + number + "\n" +
		"aliases: []\n" +
		"type: ticket\n" +
		"title: " + title + "\n" +
		"state: TODO\n" +
		"labels: []\n" +
		"epic: null\n" +
		"blocked_by: []\n" +
		"created: 2026-07-10T09:14:00+05:30\n" +
		"updated: 2026-07-10T09:14:00+05:30\n" +
		"---\n\n## Description\n"
}

func TestDiscoverWalksUp(t *testing.T) {
	root := writeStore(t, nil)
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := Discover(sub)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	// t.TempDir may sit under a symlinked path (e.g. /var -> /private/var on
	// macOS); compare resolved paths.
	got, _ := filepath.EvalSymlinks(s.Root())
	want, _ := filepath.EvalSymlinks(root)
	if got != want {
		t.Fatalf("root = %q, want %q", got, want)
	}
}

func TestDiscoverMissingIsEnvError(t *testing.T) {
	_, err := Discover(t.TempDir())
	var ce *Error
	if !errors.As(err, &ce) || ce.Code != ExitEnv {
		t.Fatalf("want ExitEnv error, got %v", err)
	}
}

func TestLoadAllSortedByULID(t *testing.T) {
	root := writeStore(t, map[string]string{
		"01J8X8Q7RZTN5Y3VXW2A9K4E7F.md": sampleItem("01J8X8Q7RZTN5Y3VXW2A9K4E7F", "KIRA-2", "second"),
		"01J8X7B1Q2W3E4R5T6Y7U8I9O0.md": sampleItem("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "first"),
		".01J8ignored.md.tmp":           "garbage",
	})
	s := &Store{root: root}
	items, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2 (temp file must be skipped)", len(items))
	}
	if items[0].ID >= items[1].ID {
		t.Fatalf("items not sorted by ULID: %s then %s", items[0].ID, items[1].ID)
	}
	snap := snapshot("KIRA", items)
	if snap.Key != "KIRA" || len(snap.Items) != 2 {
		t.Fatalf("snapshot projection wrong: %+v", snap)
	}
}

func TestLoadAllRejectsMalformed(t *testing.T) {
	root := writeStore(t, map[string]string{
		"01J8X8Q7RZTN5Y3VXW2A9K4E7F.md": "no frontmatter here",
	})
	s := &Store{root: root}
	if _, err := s.LoadAll(); err == nil {
		t.Fatal("expected LoadAll to reject a malformed file")
	}
}

func TestConfigTemplateParses(t *testing.T) {
	cfg, err := config.Parse([]byte(initConfigYAML("ACME", "acme")))
	if err != nil {
		t.Fatalf("scaffolded config must parse: %v", err)
	}
	if cfg.Project.Key != "ACME" {
		t.Fatalf("key = %q, want ACME", cfg.Project.Key)
	}
	if len(cfg.Labels.Known) != 0 || len(cfg.People.Known) != 0 {
		t.Fatalf("init must seed empty vocab, got labels=%v people=%v", cfg.Labels.Known, cfg.People.Known)
	}
	if cfg.Commit.Mode != config.CommitAuto {
		t.Fatalf("commit mode = %q, want auto", cfg.Commit.Mode)
	}
}
