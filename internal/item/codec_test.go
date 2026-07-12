package item

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readExample(tb testing.TB) string {
	tb.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "example.md"))
	if err != nil {
		tb.Fatalf("read example: %v", err)
	}
	return string(b)
}

// The canonical doc example must parse and re-serialize byte-for-byte. This is
// the keystone byte-stability contract (docs/design/03-storage-and-git.md §3).
func TestGoldenRoundTrip(t *testing.T) {
	orig := readExample(t)
	it, err := Parse(orig)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := it.Serialize(); got != orig {
		t.Fatalf("round-trip not byte-identical\n--- got ---\n%s\n--- want ---\n%s", got, orig)
	}
}

func TestParseFields(t *testing.T) {
	it, err := Parse(readExample(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checks := []struct {
		name, got, want string
	}{
		{"id", it.ID, "01J8X8Q7RZTN5Y3VXW2A9K4E7F"},
		{"number", it.Number, "KIRA-142"},
		{"type", it.Type, "ticket"},
		{"title", it.Title, "Fix race in order-book snapshot merge"},
		{"state", it.State, "IN_PROGRESS"},
		{"created", it.Created, "2026-07-10T09:14:00+05:30"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
	if it.Priority == nil || *it.Priority != "P1" {
		t.Errorf("priority = %v, want P1", it.Priority)
	}
	if it.Epic == nil || *it.Epic != "01J8X7B1Q2W3E4R5T6Y7U8I9O0" {
		t.Errorf("epic = %v", it.Epic)
	}
	if it.Estimate == nil || *it.Estimate != 3 {
		t.Errorf("estimate = %v, want 3", it.Estimate)
	}
	if len(it.Labels) != 2 || it.Labels[0] != "bug" || it.Labels[1] != "orderbook" {
		t.Errorf("labels = %v", it.Labels)
	}
	if len(it.Aliases) != 0 {
		t.Errorf("aliases = %v, want empty", it.Aliases)
	}
	if tm, err := it.CreatedTime(); err != nil || tm.Minute() != 14 {
		t.Errorf("CreatedTime = %v, %v", tm, err)
	}
}

// A single scalar edit must rewrite exactly one frontmatter line; every other
// line is byte-identical (docs/design/03-storage-and-git.md §3).
func TestSingleFieldEditOneLineDiff(t *testing.T) {
	orig := readExample(t)
	it, err := Parse(orig)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	it.State = "REVIEW"
	got := it.Serialize()

	origLines := strings.Split(orig, "\n")
	gotLines := strings.Split(got, "\n")
	if len(origLines) != len(gotLines) {
		t.Fatalf("line count changed: %d -> %d", len(origLines), len(gotLines))
	}
	var diff []int
	for i := range origLines {
		if origLines[i] != gotLines[i] {
			diff = append(diff, i)
		}
	}
	if len(diff) != 1 {
		t.Fatalf("expected exactly one changed line, got %v", diff)
	}
	if got := gotLines[diff[0]]; got != "state: REVIEW" {
		t.Fatalf("changed line = %q", got)
	}
}

func TestParseCollectsAllErrors(t *testing.T) {
	// Missing two required fields (title, state) plus a bad timestamp.
	src := "---\n" +
		"id: 01J8X8Q7RZTN5Y3VXW2A9K4E7F\n" +
		"number: KIRA-1\n" +
		"aliases: []\n" +
		"type: ticket\n" +
		"labels: []\n" +
		"epic: null\n" +
		"blocked_by: []\n" +
		"created: not-a-timestamp\n" +
		"updated: 2026-07-12T11:02:00+05:30\n" +
		"---\nbody\n"

	_, err := Parse(src)
	if err == nil {
		t.Fatal("expected error")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("want *ParseError, got %T", err)
	}
	msg := err.Error()
	for _, want := range []string{`"title"`, `"state"`, `"created"`} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message missing %s: %s", want, msg)
		}
	}
	if len(pe.Errs) < 3 {
		t.Errorf("want >=3 collected errors, got %d: %v", len(pe.Errs), pe.Errs)
	}
}

func TestParseMalformed(t *testing.T) {
	for _, src := range []string{
		"no frontmatter at all\n",
		"---\nid: x\nno closing fence\n",
	} {
		if _, err := Parse(src); err == nil {
			t.Errorf("expected error for %q", src)
		}
	}
}

func TestParseComments(t *testing.T) {
	it, err := Parse(readExample(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	cs := ParseComments(it.Body)
	if len(cs) != 1 {
		t.Fatalf("want 1 comment, got %d", len(cs))
	}
	c := cs[0]
	if c.ID != "01J8XA1F6Q2N9K3M7V0R5T8B4C" || c.Author != "shivam" ||
		c.Ts != "2026-07-11T18:30:00+05:30" ||
		c.Body != "Confirmed repro with TSan; missing acquire fence on the consumer side." {
		t.Fatalf("comment mismatch: %+v", c)
	}
}

// AppendComment must be a pure byte-suffix: the original file is an exact prefix
// of the result, and the new comment is parseable as the last block.
func TestAppendComment(t *testing.T) {
	orig := readExample(t)
	c := Comment{
		ID:     "01J8XB000000000000000000ZZ",
		Author: "alice",
		Ts:     "2026-07-12T12:00:00+05:30",
		Body:   "Second comment.\nWith two lines.",
	}
	got := AppendComment(orig, c)
	if !strings.HasPrefix(got, orig) {
		t.Fatal("original content is not a strict prefix of the result")
	}
	cs := ParseComments(got)
	if len(cs) != 2 {
		t.Fatalf("want 2 comments after append, got %d", len(cs))
	}
	if cs[1] != c {
		t.Fatalf("appended comment mismatch: %+v", cs[1])
	}
}
