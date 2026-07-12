package codec_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func readFixture(tb testing.TB, name string) string {
	tb.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		tb.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

func readExample(tb testing.TB) string { return readFixture(tb, "example.md") }

func TestGoldenRoundTrip(t *testing.T) {
	for _, name := range []string{"example.md", "legacy.md"} {
		t.Run(name, func(t *testing.T) {
			orig := readFixture(t, name)
			it, err := codec.Parse(orig)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := codec.Serialize(it); got != orig {
				t.Fatalf("round-trip not byte-identical\n--- got ---\n%s\n--- want ---\n%s", got, orig)
			}
		})
	}
}

func TestParseFields(t *testing.T) {
	it, err := codec.Parse(readExample(t))
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
	optChecks := []struct {
		name string
		got  *string
		want string
	}{
		{"subtype", it.Subtype, "bug"},
		{"priority", it.Priority, "P1"},
		{"rank", it.Rank, "0|hzzzzz:"},
		{"sprint", it.Sprint, "2026-S14"},
		{"due", it.Due, "2026-07-20"},
	}
	for _, c := range optChecks {
		if c.got == nil || *c.got != c.want {
			t.Errorf("%s = %v, want %q", c.name, c.got, c.want)
		}
	}
	if it.Resolution != nil {
		t.Errorf("resolution = %q, want absent", *it.Resolution)
	}
	wantLinks := map[string][]string{
		datamodel.LinkRelates:     {"01J8XB3K9P0Q2R4S6T8V0W1X2Y"},
		datamodel.LinkDuplicateOf: {"01J8XC4M0N1P2Q3R4S5T6U7V8W"},
	}
	if !reflect.DeepEqual(it.Links, wantLinks) {
		t.Errorf("links = %v, want %v", it.Links, wantLinks)
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

func TestSingleFieldEditOneLineDiff(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(*datamodel.Item)
		wantLine string
	}{
		{"state", func(it *datamodel.Item) { it.State = "REVIEW" }, "state: REVIEW"},
		{"sprint", func(it *datamodel.Item) { s := "2026-S15"; it.Sprint = &s }, "sprint: 2026-S15"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orig := readExample(t)
			it, err := codec.Parse(orig)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			tc.mutate(it)
			got := codec.Serialize(it)

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
			if got := gotLines[diff[0]]; got != tc.wantLine {
				t.Fatalf("changed line = %q", got)
			}
		})
	}
}

func TestLinksCanonicalization(t *testing.T) {
	src := "---\n" +
		"id: 01J8X8Q7RZTN5Y3VXW2A9K4E7F\n" +
		"number: KIRA-1\n" +
		"aliases: []\n" +
		"type: ticket\n" +
		"title: \"t\"\n" +
		"state: TODO\n" +
		"labels: []\n" +
		"epic: null\n" +
		"blocked_by: []\n" +
		"links:\n" +
		"  duplicate_of: [01J8XC4M0N1P2Q3R4S5T6U7V8W]\n" +
		"  relates: []\n" +
		"created: 2026-07-10T09:14:00+05:30\n" +
		"updated: 2026-07-12T11:02:00+05:30\n" +
		"---\nbody\n"
	it, err := codec.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := map[string][]string{datamodel.LinkDuplicateOf: {"01J8XC4M0N1P2Q3R4S5T6U7V8W"}}
	if !reflect.DeepEqual(it.Links, want) {
		t.Fatalf("links = %v, want %v", it.Links, want)
	}
	if out := codec.Serialize(it); !strings.Contains(out, "links:\n  duplicate_of: [01J8XC4M0N1P2Q3R4S5T6U7V8W]\ncreated:") {
		t.Fatalf("canonical links emission wrong:\n%s", out)
	}

	if _, err := codec.Parse(strings.Replace(src, "duplicate_of:", "mystery:", 1)); err == nil {
		t.Fatal("unknown link type: expected rejection")
	}

	it.Links = nil
	if out := codec.Serialize(it); strings.Contains(out, "links") {
		t.Fatalf("empty links must emit no line:\n%s", out)
	}
}

func TestDueParseIsShapeOnly(t *testing.T) {
	src := strings.Replace(readExample(t), "due: 2026-07-20", "due: not-a-date", 1)
	it, err := codec.Parse(src)
	if err != nil {
		t.Fatalf("invalid due must still parse: %v", err)
	}
	if it.Due == nil || *it.Due != "not-a-date" {
		t.Fatalf("due = %v, want the raw invalid value", it.Due)
	}
	if got := codec.Serialize(it); got != src {
		t.Fatalf("invalid due must round-trip\n--- got ---\n%s", got)
	}
}

func TestParseCollectsAllErrors(t *testing.T) {
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

	_, err := codec.Parse(src)
	if err == nil {
		t.Fatal("expected error")
	}
	var pe *datamodel.ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("want *datamodel.ParseError, got %T", err)
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

func TestBodyHorizontalRuleIsNotAFence(t *testing.T) {
	src := strings.Replace(readExample(t), "## Description", "---\n\n## Description", 1)
	it, err := codec.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !strings.Contains(it.Body, "---\n\n## Description") {
		t.Fatalf("body lost the horizontal rule:\n%s", it.Body)
	}
	if got := codec.Serialize(it); got != src {
		t.Fatalf("round-trip with body --- not byte-identical\n--- got ---\n%s", got)
	}
}

func TestParseMalformed(t *testing.T) {
	for _, src := range []string{
		"no frontmatter at all\n",
		"---\nid: x\nno closing fence\n",
	} {
		if _, err := codec.Parse(src); err == nil {
			t.Errorf("expected error for %q", src)
		}
	}
}

func TestParseComments(t *testing.T) {
	it, err := codec.Parse(readExample(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	cs := codec.ParseComments(it.Body)
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

func TestAppendComment(t *testing.T) {
	orig := readExample(t)
	c := datamodel.Comment{
		ID:     "01J8XB000000000000000000ZZ",
		Author: "alice",
		Ts:     "2026-07-12T12:00:00+05:30",
		Body:   "Second comment.\nWith two lines.",
	}
	got := codec.AppendComment(orig, c)
	if !strings.HasPrefix(got, orig) {
		t.Fatal("original content is not a strict prefix of the result")
	}
	cs := codec.ParseComments(got)
	if len(cs) != 2 {
		t.Fatalf("want 2 comments after append, got %d", len(cs))
	}
	if cs[1] != c {
		t.Fatalf("appended comment mismatch: %+v", cs[1])
	}
}
