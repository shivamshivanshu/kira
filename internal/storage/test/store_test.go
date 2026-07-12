package storage_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func writeStore(t *testing.T, tickets map[string]string) string {
	t.Helper()
	root := t.TempDir()
	ticketsDir := filepath.Join(root, ".kira", "tickets")
	if err := os.MkdirAll(ticketsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range tickets {
		if err := os.WriteFile(filepath.Join(ticketsDir, name), []byte(content), 0o644); err != nil {
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
	s, err := storage.Discover(sub)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	got, _ := filepath.EvalSymlinks(s.Root())
	want, _ := filepath.EvalSymlinks(root)
	if got != want {
		t.Fatalf("root = %q, want %q", got, want)
	}
}

func TestDiscoverMissingIsEnvError(t *testing.T) {
	_, err := storage.Discover(t.TempDir())
	var ce *errx.Error
	if !errors.As(err, &ce) || ce.Code != errx.ExitEnv {
		t.Fatalf("want errx.ExitEnv error, got %v", err)
	}
}
