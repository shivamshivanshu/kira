package rgx_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/rgx"
)

func TestParseLine(t *testing.T) {
	cases := []struct {
		name string
		line string
		want rgx.Line
		ok   bool
	}{
		{"match line", ".kira/tickets/01J8.md:14:the snapshot merge", rgx.Line{Path: ".kira/tickets/01J8.md", IsMatch: true, LineNo: 14, Text: "the snapshot merge"}, true},
		{"context line", ".kira/tickets/01J8.md-13-context line", rgx.Line{Path: ".kira/tickets/01J8.md", IsMatch: false, LineNo: 13, Text: "context line"}, true},
		{"separator", "--", rgx.Line{}, false},
		{"line number overflows int", ".kira/tickets/01J8.md:99999999999999999999:x", rgx.Line{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := rgx.ParseLine(c.line)
			if ok != c.ok {
				t.Fatalf("%q: ok=%v, want %v", c.line, ok, c.ok)
			}
			if got != c.want {
				t.Fatalf("%q: parsed %+v, want %+v", c.line, got, c.want)
			}
		})
	}
}

func TestSearch(t *testing.T) {
	if !rgx.Installed() {
		t.Skip("rg not installed")
	}

	dir := t.TempDir()
	itemsDir := filepath.Join(dir, "items")
	if err := os.MkdirAll(itemsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(itemsDir, "a.md"), []byte("alpha\nbeta race\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("match", func(t *testing.T) {
		lines, err := rgx.Search(dir, []string{"race"}, "items")
		if err != nil {
			t.Fatal(err)
		}
		if len(lines) != 1 || !lines[0].IsMatch || lines[0].LineNo != 2 {
			t.Fatalf("got %+v", lines)
		}
	})

	t.Run("enforced flags win over a conflicting passthru flag", func(t *testing.T) {
		lines, err := rgx.Search(dir, []string{"--heading", "race"}, "items")
		if err != nil {
			t.Fatal(err)
		}
		if len(lines) != 1 || lines[0].Path == "" {
			t.Fatalf("--heading passthru must not defeat the enforced --no-heading: got %+v", lines)
		}
	})

	t.Run("partial matches survive an error exit", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("running as root: permission bits don't block reads")
		}
		bad := filepath.Join(itemsDir, "unreadable.md")
		if err := os.WriteFile(bad, []byte("race\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(bad, 0o000); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(bad, 0o644) }()

		lines, err := rgx.Search(dir, []string{"race"}, "items")
		if err == nil {
			t.Fatal("want an error from the unreadable file")
		}
		if len(lines) == 0 {
			t.Fatalf("want the match from a.md preserved alongside the error, got none")
		}
	})
}
