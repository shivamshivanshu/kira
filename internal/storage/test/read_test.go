package storage_test

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/storage"
)

func TestLoadAllSortedByULID(t *testing.T) {
	root := writeStore(t, map[string]string{
		"01J8X8Q7RZTN5Y3VXW2A9K4E7F.md": sampleItem("01J8X8Q7RZTN5Y3VXW2A9K4E7F", "KIRA-2", "second"),
		"01J8X7B1Q2W3E4R5T6Y7U8I9O0.md": sampleItem("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "first"),
		".01J8ignored.md.tmp":           "garbage",
	})
	s := storage.New(root)
	items, _, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2 (temp file must be skipped)", len(items))
	}
	if items[0].ID >= items[1].ID {
		t.Fatalf("items not sorted by ULID: %s then %s", items[0].ID, items[1].ID)
	}
	snap := storage.Snapshot("KIRA", items)
	if snap.Key != "KIRA" || len(snap.Items) != 2 {
		t.Fatalf("snapshot projection wrong: %+v", snap)
	}
}

func TestLoadAllSkipsMalformed(t *testing.T) {
	root := writeStore(t, map[string]string{
		"01J8X7B1Q2W3E4R5T6Y7U8I9O0.md": sampleItem("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "first"),
		"01J8X8Q7RZTN5Y3VXW2A9K4E7F.md": "no frontmatter here",
	})
	s := storage.New(root)
	items, warnings, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll must not fail on a malformed file: %v", err)
	}
	if len(items) != 1 || items[0].Number != "KIRA-1" {
		t.Fatalf("healthy ticket should still load, got %d items", len(items))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 skip warning, got %v", warnings)
	}
}
