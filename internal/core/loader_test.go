package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func hasWarning(notes []datamodel.Warning, code datamodel.WarnCode) bool {
	for _, n := range notes {
		if n.Code == code {
			return true
		}
	}
	return false
}

func findWarning(notes []datamodel.Warning, code datamodel.WarnCode) (datamodel.Warning, bool) {
	for _, n := range notes {
		if n.Code == code {
			return n, true
		}
	}
	return datamodel.Warning{}, false
}

func TestItemsIndexFallbackAndLinear(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".kira", "tickets"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := newStore(dir)
	cfg := config.Default()
	cfg.Project.Key = "KIRA"
	it := eventTicket()
	if _, err := s.fs().WriteItem(it); err != nil {
		t.Fatalf("writeItem: %v", err)
	}

	fallback, err := s.read(cfg, loadOpts{useIndex: true})
	if err != nil {
		t.Fatalf("items useIndex on a non-git store: %v", err)
	}
	if len(fallback.items) != 1 || fallback.items[0].ID != it.ID {
		t.Fatalf("fallback items = %v, want the one written item", fallback.items)
	}
	warn, ok := findWarning(fallback.notes, datamodel.WarnIndexFallback)
	if !ok {
		t.Fatalf("index unavailable must surface a fallback warning, got %v", fallback.notes)
	}
	if len(warn.Args) == 0 || warn.Args[0] == "" {
		t.Fatalf("fallback warning must carry the swallowed index error, got %+v", warn)
	}

	linear, err := s.read(cfg, loadOpts{useIndex: false})
	if err != nil {
		t.Fatalf("items linear: %v", err)
	}
	if len(linear.items) != 1 || linear.items[0].ID != it.ID {
		t.Fatalf("linear items = %v, want the one written item", linear.items)
	}
	if len(linear.notes) != 0 {
		t.Fatalf("linear load must not warn, got %v", linear.notes)
	}
}

func TestItemsIndexHitNoFallback(t *testing.T) {
	s := eventRepo(t)
	cfg := config.Default()
	it := eventTicket()
	commitState(t, s, it, "TODO", "2026-01-05")

	ld, err := s.read(cfg, loadOpts{useIndex: true})
	if err != nil {
		t.Fatalf("items useIndex on a healthy repo: %v", err)
	}
	if len(ld.items) != 1 || ld.items[0].ID != it.ID {
		t.Fatalf("index items = %v, want the committed item", ld.items)
	}
	if hasWarning(ld.notes, datamodel.WarnIndexFallback) {
		t.Fatalf("a healthy index must not fall back, got %v", ld.notes)
	}
}
