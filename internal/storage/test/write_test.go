package storage_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/storage"
)

func TestWriteItemRawRemovesTempOnFailure(t *testing.T) {
	root := writeStore(t, nil)
	s := storage.New(root)
	ulid := "01J8X7B1Q2W3E4R5T6Y7U8I9O0"

	ticketsDir := filepath.Join(root, ".kira", "tickets")
	if err := os.Mkdir(filepath.Join(ticketsDir, ulid+".md"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := s.WriteItemRaw(ulid, "body"); err == nil {
		t.Fatal("WriteItemRaw must fail when destination path is a directory")
	}

	entries, err := os.ReadDir(ticketsDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temp file leaked after failed write: %s", e.Name())
		}
	}
}

func TestWriteItemRawLeavesNoTempOnSuccess(t *testing.T) {
	root := writeStore(t, nil)
	s := storage.New(root)
	ulid := "01J8X8Q7RZTN5Y3VXW2A9K4E7F"

	if _, err := s.WriteItemRaw(ulid, sampleItem(ulid, "KIRA-1", "first")); err != nil {
		t.Fatalf("WriteItemRaw: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(root, ".kira", "tickets"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 1 || names[0] != ulid+".md" {
		t.Fatalf("want exactly %s.md, got %v", ulid, names)
	}
}
