package query

import (
	"sort"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
)

func strp(s string) *string { return &s }

// fixture builds a small item set plus a resolver over it. The epic KIRA-100 is
// the parent of KIRA-1. ULIDs are minted so the resolver's number/prefix rules
// are exercised realistically.
func fixture() (items []*item.Item, resolver *id.Resolver, cfg *config.Config) {
	cfg = config.Default()
	epicID := id.Mint().String()
	it1ID := id.Mint().String()
	it2ID := id.Mint().String()
	it3ID := id.Mint().String()

	epic := &item.Item{
		ID: epicID, Number: "KIRA-100", Type: item.TypeEpic, Title: "Big epic",
		State: "ACTIVE", Created: "2026-07-01T00:00:00Z", Updated: "2026-07-01T00:00:00Z",
	}
	it1 := &item.Item{
		ID: it1ID, Number: "KIRA-1", Type: item.TypeTicket, Title: "Fix race in snapshot",
		State: "IN_PROGRESS", Owner: strp("shivam"), Priority: strp("P1"),
		Labels: []string{"bug"}, Epic: strp(epicID),
		Created: "2026-07-05T00:00:00Z", Updated: "2026-07-06T00:00:00Z",
	}
	it2 := &item.Item{
		ID: it2ID, Number: "KIRA-2", Type: item.TypeTicket, Title: "Perf tuning",
		State: "TODO", Owner: strp("alice"), Labels: []string{"perf"},
		Created: "2026-07-10T00:00:00Z", Updated: "2026-07-10T00:00:00Z",
	}
	it3 := &item.Item{
		ID: it3ID, Number: "KIRA-3", Type: item.TypeTicket, Title: "Done thing",
		State: "DONE", Owner: strp("shivam"), Labels: []string{"bug", "perf"},
		Created: "2026-06-01T00:00:00Z", Updated: "2026-06-02T00:00:00Z",
	}
	items = []*item.Item{epic, it1, it2, it3}

	snap := id.Snapshot{Key: "KIRA"}
	for _, it := range items {
		snap.Items = append(snap.Items, id.Item{ULID: it.ID, Number: it.Number, Aliases: it.Aliases})
	}
	return items, id.NewResolver(snap), cfg
}

// matchNums compiles expr and returns the display numbers it matches, sorted.
func matchNums(t *testing.T, expr string, items []*item.Item, r *id.Resolver, cfg *config.Config) []string {
	t.Helper()
	pred, err := Compile(expr, r)
	if err != nil {
		t.Fatalf("Compile(%q): %v", expr, err)
	}
	var out []string
	for _, it := range items {
		if pred(it, cfg) {
			out = append(out, it.Number)
		}
	}
	sort.Strings(out)
	return out
}

func TestEval(t *testing.T) {
	items, r, cfg := fixture()
	tests := []struct {
		expr string
		want []string
	}{
		{"state=IN_PROGRESS", []string{"KIRA-1"}},
		{"state!=IN_PROGRESS", []string{"KIRA-100", "KIRA-2", "KIRA-3"}},
		{"owner=shivam", []string{"KIRA-1", "KIRA-3"}},
		{"owner!=shivam", []string{"KIRA-100", "KIRA-2"}}, // epic owner is nil -> ""
		{"label=bug", []string{"KIRA-1", "KIRA-3"}},
		{"label=perf", []string{"KIRA-2", "KIRA-3"}},
		{"label=bug AND label=perf", []string{"KIRA-3"}},
		{"label!=bug", []string{"KIRA-100", "KIRA-2"}},
		{"category=doing", []string{"KIRA-1", "KIRA-100"}}, // IN_PROGRESS and epic ACTIVE
		{"category=done", []string{"KIRA-3"}},
		{"type=epic", []string{"KIRA-100"}},
		{"type=ticket", []string{"KIRA-1", "KIRA-2", "KIRA-3"}},
		{"priority=P1", []string{"KIRA-1"}},
		{"epic=KIRA-100", []string{"KIRA-1"}},
		{"NOT epic=KIRA-100", []string{"KIRA-100", "KIRA-2", "KIRA-3"}},
		{"created>2026-06-15", []string{"KIRA-1", "KIRA-100", "KIRA-2"}},
		{"created<2026-07-01", []string{"KIRA-3"}},
		{"created>=2026-07-01", []string{"KIRA-1", "KIRA-100", "KIRA-2"}},
		{"updated<=2026-07-01", []string{"KIRA-100", "KIRA-3"}},
		{"race", []string{"KIRA-1"}},       // title substring
		{"RACE", []string{"KIRA-1"}},       // case-insensitive term
		{`"fix race"`, []string{"KIRA-1"}}, // quoted multi-word term
		{"owner=shivam AND label=perf", []string{"KIRA-3"}},
		{"owner=alice OR label=bug", []string{"KIRA-1", "KIRA-2", "KIRA-3"}},
		{"category=doing AND NOT owner=alice", []string{"KIRA-1", "KIRA-100"}},
		{"epic=KIRA-100 AND created>2026-07-01", []string{"KIRA-1"}},
	}
	for _, tc := range tests {
		got := matchNums(t, tc.expr, items, r, cfg)
		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Errorf("%q matched %v, want %v", tc.expr, got, tc.want)
		}
	}
}

// TestCompileEpicUnresolved reports a positioned error when an epic value
// resolves to no item.
func TestCompileEpicUnresolved(t *testing.T) {
	_, r, _ := fixture()
	_, err := Compile("epic=KIRA-999", r)
	qe, ok := err.(*Error)
	if !ok {
		t.Fatalf("err = %v, want *Error", err)
	}
	if qe.Pos != len("epic=") {
		t.Errorf("pos = %d, want %d", qe.Pos, len("epic="))
	}
}

// TestMatch builds single-field predicates the way the flat CLI filters do.
func TestMatch(t *testing.T) {
	items, r, cfg := fixture()
	pred, err := Match("owner", "shivam", r)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	var n int
	for _, it := range items {
		if pred(it, cfg) {
			n++
		}
	}
	if n != 2 {
		t.Errorf("Match owner=shivam matched %d, want 2", n)
	}
	// epic value is resolved through the same engine.
	if _, err := Match("epic", "KIRA-100", r); err != nil {
		t.Errorf("Match epic=KIRA-100: %v", err)
	}
	// date fields are ordering-only, not equality filters.
	if _, err := Match("created", "2026-01-01", r); err == nil {
		t.Errorf("Match on a date field should be rejected")
	}
}
