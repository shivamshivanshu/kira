package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func writeStore(t *testing.T, tickets map[string]string) string {
	t.Helper()
	root := t.TempDir()
	kira := filepath.Join(root, ".kira")
	if err := os.MkdirAll(filepath.Join(kira, "tickets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kira, "config.yaml"), []byte(seedConfigYAML(t)), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, content := range tickets {
		if err := os.WriteFile(filepath.Join(kira, "tickets", name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func itemWithBody(ulid, number, title, body string) string {
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
		"---\n\n" + body
}

func findIn(t *testing.T, tickets map[string]string, args FindArgs) (*datamodel.FindResult, error) {
	t.Helper()
	root := writeStore(t, tickets)
	s := newStore(root)
	items, _, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	rows, err := s.findInItems(args, items)
	if err != nil {
		return nil, err
	}
	res := NewFindResult(rows)
	return &res, nil
}

func TestFindMatch(t *testing.T) {
	res, err := findIn(t, map[string]string{
		"01J8X7B1Q2W3E4R5T6Y7U8I9O0.md": itemWithBody("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "first", "The snapshot merge drops updates.\n"),
		"01J8X8Q7RZTN5Y3VXW2A9K4E7F.md": itemWithBody("01J8X8Q7RZTN5Y3VXW2A9K4E7F", "KIRA-2", "second", "Unrelated body.\n"),
	}, FindArgs{Pattern: "snapshot"})
	if err != nil {
		t.Fatalf("findInItems: %v", err)
	}
	if len(res.Matches) != 1 {
		t.Fatalf("got %d matches, want 1: %+v", len(res.Matches), res.Matches)
	}
	m := res.Matches[0]
	if m.Number != "KIRA-1" || m.ID != "01J8X7B1Q2W3E4R5T6Y7U8I9O0" {
		t.Fatalf("match mapped to wrong item: %+v", m)
	}
	if m.Line != 15 {
		t.Fatalf("line = %d, want 15 (first body line)", m.Line)
	}
}

func TestFindNoMatch(t *testing.T) {
	res, err := findIn(t, map[string]string{
		"01J8X7B1Q2W3E4R5T6Y7U8I9O0.md": itemWithBody("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "first", "nothing here\n"),
	}, FindArgs{Pattern: "absent"})
	if err != nil {
		t.Fatalf("findInItems: %v", err)
	}
	if len(res.Matches) != 0 {
		t.Fatalf("want no matches, got %+v", res.Matches)
	}
	if res.Matches == nil {
		t.Fatal("matches must be non-nil empty for JSON []")
	}
}

func TestFindRegexErrorIsUserError(t *testing.T) {
	_, err := findIn(t, map[string]string{
		"01J8X7B1Q2W3E4R5T6Y7U8I9O0.md": itemWithBody("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "first", "x\n"),
	}, FindArgs{Pattern: "["})
	var ce *errx.Error
	if !errors.As(err, &ce) || ce.Code != errx.ExitUser {
		t.Fatalf("bad regex must be a user error (exit 1), got %v", err)
	}
}

func TestFindIgnoreCaseAndWord(t *testing.T) {
	tickets := map[string]string{
		"01J8X7B1Q2W3E4R5T6Y7U8I9O0.md": itemWithBody("01J8X7B1Q2W3E4R5T6Y7U8I9O0", "KIRA-1", "first", "Race in the orderbook.\nembraced the change\n"),
	}
	res, err := findIn(t, tickets, FindArgs{Pattern: "race"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Matches) != 1 || res.Matches[0].Line != 16 {
		t.Fatalf("case-sensitive should match only 'embraced' line 16, got %+v", res.Matches)
	}
	res, err = findIn(t, tickets, FindArgs{Pattern: "race", IgnoreCase: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Matches) != 2 {
		t.Fatalf("-i should match both lines, got %+v", res.Matches)
	}
	res, err = findIn(t, tickets, FindArgs{Pattern: "race", IgnoreCase: true, Word: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Matches) != 1 || res.Matches[0].Line != 15 {
		t.Fatalf("-w should match only the standalone word on line 15, got %+v", res.Matches)
	}
}

func TestFindEmptyPatternIsUserError(t *testing.T) {
	_, err := findIn(t, nil, FindArgs{Pattern: ""})
	var ce *errx.Error
	if !errors.As(err, &ce) || ce.Code != errx.ExitUser {
		t.Fatalf("empty pattern must be a user error, got %v", err)
	}
}
