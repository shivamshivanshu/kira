package core

import (
	"errors"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
)

func strptr(s string) *string { return &s }

func mkItem(number, ulid string, rank, priority *string) *item.Item {
	return &item.Item{ID: ulid, Number: number, Type: item.TypeTicket, Rank: rank, Priority: priority}
}

func numbers(items []*item.Item) string {
	nums := make([]string, len(items))
	for i, it := range items {
		nums[i] = it.Number
	}
	return strings.Join(nums, ",")
}

func TestSortByPrecedence(t *testing.T) {
	cfg := config.Default() // priorities P0..P3
	items := []*item.Item{
		mkItem("KIRA-1", "01A", nil, nil),
		mkItem("KIRA-2", "01B", strptr("mm"), nil),
		mkItem("KIRA-3", "01C", nil, strptr("P2")),
		mkItem("KIRA-4", "01D", strptr("aa"), strptr("P3")),
		mkItem("KIRA-5", "01E", nil, strptr("P0")),
		mkItem("KIRA-6", "01F", nil, strptr("P2")),
	}
	sortByPrecedence(cfg, items)
	// ranked first by rank (aa < mm, priority irrelevant), then priority
	// config order (P0, P2, P2 tie by number), unranked+unprioritized last.
	want := "KIRA-4,KIRA-2,KIRA-5,KIRA-3,KIRA-6,KIRA-1"
	if got := numbers(items); got != want {
		t.Errorf("precedence order = %s, want %s", got, want)
	}
}

func TestSortByPrecedenceLegacyDegradation(t *testing.T) {
	cfg := config.Default()
	cfg.Priorities = nil
	items := []*item.Item{
		mkItem("KIRA-3", "01C", nil, nil),
		mkItem("KIRA-1", "01A", nil, nil),
		mkItem("HASH-x", "01Z", nil, nil), // unparsable number falls back to string order
		mkItem("KIRA-2", "01B", nil, nil),
	}
	sortByPrecedence(cfg, items)
	legacy := []*item.Item{items[0], items[1], items[2], items[3]}
	sortByKey(legacy, func(it *item.Item) id.SortKey { return id.NewSortKey(it.Number, it.ID) })
	if got, want := numbers(items), numbers(legacy); got != want {
		t.Errorf("degraded order = %s, want legacy %s", got, want)
	}
	// A priority set on an item is inert without the vocabulary.
	items[0].Priority = strptr("high")
	sortByPrecedence(cfg, items)
	if got, want := numbers(items), numbers(legacy); got != want {
		t.Errorf("free-form priority perturbed order: %s, want %s", got, want)
	}
}

func listFixture(t *testing.T) (*Store, *config.Config) {
	t.Helper()
	root := initGitRepo(t)
	if _, err := Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, _ := Discover(root)
	cfg, _ := s.Config()
	cfg.Filters = map[string]string{
		"mine":    "owner=shivam",
		"groomed": "rank IS NOT EMPTY ORDER BY rank",
	}
	cfg.Sprints = []config.Sprint{{Key: "2026-S14", Name: "Sprint 14", Start: "2026-07-13", End: "2026-07-26"}}
	for _, c := range []CreateOpts{
		{Type: item.TypeTicket, Title: "alpha", Owner: "shivam", Priority: "P2", NoEdit: true},
		{Type: item.TypeTicket, Title: "beta", Owner: "alice", Priority: "P0", Sprint: "2026-S14", NoEdit: true},
		{Type: item.TypeTicket, Title: "gamma", Owner: "shivam", Rank: "mm", Due: "2026-07-20", NoEdit: true},
	} {
		if _, err := s.Create(cfg, c); err != nil {
			t.Fatalf("create %s: %v", c.Title, err)
		}
	}
	return s, cfg
}

func listNumbers(res *ListResult) string {
	nums := make([]string, len(res.Items))
	for i, it := range res.Items {
		nums[i] = it.Number
	}
	return strings.Join(nums, ",")
}

func TestListDefaultPrecedenceAndOrderBy(t *testing.T) {
	s, cfg := listFixture(t)

	// Default: ranked KIRA-3 first, then P0 before P2.
	res, err := s.List(cfg, ListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got, want := listNumbers(res), "KIRA-3,KIRA-2,KIRA-1"; got != want {
		t.Errorf("default order = %s, want %s", got, want)
	}

	// ORDER BY overrides the precedence; null due sorts last both directions.
	res, err = s.List(cfg, ListOpts{Query: "type=ticket ORDER BY priority"})
	if err != nil {
		t.Fatalf("List order by priority: %v", err)
	}
	if got, want := listNumbers(res), "KIRA-2,KIRA-1,KIRA-3"; got != want {
		t.Errorf("ORDER BY priority = %s, want %s", got, want)
	}
	res, err = s.List(cfg, ListOpts{Query: "type=ticket ORDER BY due desc"})
	if err != nil {
		t.Fatalf("List order by due desc: %v", err)
	}
	// Only KIRA-3 has a due date; the dueless keep the default precedence.
	if got, want := listNumbers(res), "KIRA-3,KIRA-2,KIRA-1"; got != want {
		t.Errorf("ORDER BY due desc = %s, want %s", got, want)
	}
}

func TestListFlagFilters(t *testing.T) {
	s, cfg := listFixture(t)

	res, err := s.List(cfg, ListOpts{Priority: "P0"})
	if err != nil {
		t.Fatalf("List --priority: %v", err)
	}
	if got := listNumbers(res); got != "KIRA-2" {
		t.Errorf("--priority P0 = %s, want KIRA-2", got)
	}

	res, err = s.List(cfg, ListOpts{Sprint: "2026-S14"})
	if err != nil {
		t.Fatalf("List --sprint: %v", err)
	}
	if got := listNumbers(res); got != "KIRA-2" {
		t.Errorf("--sprint 2026-S14 = %s, want KIRA-2", got)
	}

	// --sprint active with no pointer set: empty result plus the stderr note.
	res, err = s.List(cfg, ListOpts{Sprint: "active"})
	if err != nil {
		t.Fatalf("List --sprint active: %v", err)
	}
	if res.Count != 0 {
		t.Errorf("--sprint active with no active sprint matched %d, want 0", res.Count)
	}
	if len(res.StderrNotes) != 1 {
		t.Errorf("notes = %v, want the no-active-sprint note", res.StderrNotes)
	}
}

func TestListSavedFilters(t *testing.T) {
	s, cfg := listFixture(t)

	res, err := s.List(cfg, ListOpts{Filter: "mine"})
	if err != nil {
		t.Fatalf("List --filter mine: %v", err)
	}
	if got := listNumbers(res); got != "KIRA-3,KIRA-1" {
		t.Errorf("--filter mine = %s, want KIRA-3,KIRA-1", got)
	}

	// Extra flags AND onto the expanded filter.
	res, err = s.List(cfg, ListOpts{Filter: "mine", Priority: "P2"})
	if err != nil {
		t.Fatalf("List --filter mine --priority: %v", err)
	}
	if got := listNumbers(res); got != "KIRA-1" {
		t.Errorf("--filter mine --priority P2 = %s, want KIRA-1", got)
	}

	// A filter may carry the ORDER BY.
	res, err = s.List(cfg, ListOpts{Filter: "groomed"})
	if err != nil {
		t.Fatalf("List --filter groomed: %v", err)
	}
	if got := listNumbers(res); got != "KIRA-3" {
		t.Errorf("--filter groomed = %s, want KIRA-3", got)
	}

	// Two ORDER BY clauses (filter + query) are rejected.
	if _, err := s.List(cfg, ListOpts{Filter: "groomed", Query: "a ORDER BY due"}); err == nil {
		t.Error("filter+query each with ORDER BY should be rejected")
	}

	// Unknown name is a user error listing the configured names.
	_, err = s.List(cfg, ListOpts{Filter: "nope"})
	var ce *Error
	if !errors.As(err, &ce) || ce.Code != ExitUser {
		t.Fatalf("unknown filter err = %v, want exit-1 user error", err)
	}
	if msg := err.Error(); !strings.Contains(msg, "groomed") || !strings.Contains(msg, "mine") {
		t.Errorf("unknown filter message %q should list configured names", msg)
	}
}

func TestFiltersView(t *testing.T) {
	cfg := config.Default()
	cfg.Filters = map[string]string{"b": "owner=x", "a": "label=y"}
	res := Filters(cfg)
	if len(res.Filters) != 2 || res.Filters[0].Name != "a" || res.Filters[1].Name != "b" {
		t.Errorf("Filters = %+v, want name-sorted [a b]", res.Filters)
	}
}
