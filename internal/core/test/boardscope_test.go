package core_test

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/query"
)

func scopedQueryFixture(t *testing.T) (*core.Store, *datamodel.Config) {
	t.Helper()
	s, cfg := newStore(t)
	cfg.Commit.Mode = datamodel.CommitManual
	cfg.Boards = []datamodel.Board{{Key: "KIRA", Name: "Kira", Default: true}, {Key: "OPS", Name: "Operations"}}
	cfg.Sprints = []datamodel.Sprint{
		{Key: "s1", Name: "First", Start: "2026-08-01", End: "2026-08-14"},
		{Key: "s2", Name: "Second", Start: "2026-08-15", End: "2026-08-28"},
	}
	cfg.Filters = map[string]string{"kept": "label=keep", "ordered": "label=keep ORDER BY rank DESC"}
	for _, opts := range []core.CreateOpts{
		{Title: "BY created alpha", Board: "KIRA", Sprint: "s1", Owner: "ORDER", Labels: []string{"keep"}, Rank: "b"},
		{Title: "BY created beta", Board: "KIRA", Sprint: "s1", Owner: "ORDER", Labels: []string{"keep"}, Rank: "a"},
		{Title: "BY created other board", Board: "OPS", Sprint: "s1", Owner: "ORDER", Labels: []string{"keep"}, Rank: "aa"},
		{Title: "BY created other sprint", Board: "KIRA", Sprint: "s2", Owner: "ORDER", Labels: []string{"keep"}, Rank: "c"},
		{Title: "BY created other label", Board: "KIRA", Sprint: "s1", Owner: "other", Labels: []string{"skip"}, Rank: "0"},
		{Title: "Different title", Board: "KIRA", Sprint: "s1", Owner: "ORDER", Labels: []string{"keep"}, Rank: "0"},
	} {
		opts.Type, opts.NoEdit = datamodel.TypeTicket, true
		if _, err := s.Create(cfg, opts); err != nil {
			t.Fatal(err)
		}
	}
	return s, cfg
}

func TestBoardAndSprintScopesConjoinWithQueriesAndSavedFilters(t *testing.T) {
	t.Parallel()
	s, cfg := scopedQueryFixture(t)
	for _, tc := range []struct {
		name, filter, query, want string
	}{
		{"query order", "kept", "(owner=ORDER OR owner=other) AND BY AND created ORDER BY rank", "KIRA-2,KIRA-1"},
		{"saved filter order", "ordered", "owner=ORDER BY created", "KIRA-1,KIRA-2"},
		{"default order", "kept", "owner=ORDER BY created", "KIRA-2,KIRA-1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			list, err := s.List(cfg, core.ListOpts{Board: "kira", Sprint: "s1", Filter: tc.filter, Query: tc.query})
			if err != nil {
				t.Fatal(err)
			}
			if got := listNumbers(list); got != tc.want {
				t.Errorf("list scope = %s, want %s", got, tc.want)
			}
			board, err := s.Board(cfg, core.BoardOpts{Board: "kira", Sprint: "s1", Filter: tc.filter, Query: tc.query})
			if err != nil {
				t.Fatal(err)
			}
			column := columnByState(board, "TODO")
			if got := listNumbers(&datamodel.ListResult{Items: column.Items}); got != tc.want || column.Count != 2 {
				t.Errorf("board scope = %s, count=%d, want %s and 2", got, column.Count, tc.want)
			}
		})
	}
	for _, read := range []func() error{
		func() error {
			_, err := s.List(cfg, core.ListOpts{Board: "KIRA", Sprint: "s1", Filter: "ordered", Query: "owner=ORDER ORDER BY rank"})
			return err
		},
		func() error {
			_, err := s.Board(cfg, core.BoardOpts{Board: "KIRA", Sprint: "s1", Filter: "ordered", Query: "owner=ORDER ORDER BY rank"})
			return err
		},
	} {
		if err := read(); err == nil || !strings.Contains(err.Error(), "only one ORDER BY") {
			t.Errorf("scoping must preserve duplicate-order validation: %v", err)
		}
	}
}

func TestScopedQueryPreservesOrderWordAsOwner(t *testing.T) {
	t.Parallel()
	const expr = "owner=ORDER BY created"
	parsed, err := query.Parse(expr)
	if err != nil || parsed.Order != nil {
		t.Fatalf("ORDER is an owner value here, not a sorting clause: %+v %v", parsed, err)
	}
	s, cfg := scopedQueryFixture(t)
	if _, err := s.SprintActivate(cfg, "s1"); err != nil {
		t.Fatal(err)
	}
	board, err := s.Board(cfg, core.BoardOpts{Board: "KIRA", Sprint: "active", Query: expr})
	if err != nil {
		t.Fatal(err)
	}
	if got := listNumbers(&datamodel.ListResult{Items: columnByState(board, "TODO").Items}); got != "KIRA-2,KIRA-1" {
		t.Fatalf("active sprint and board altered query semantics: %s", got)
	}
	all, err := s.Board(cfg, core.BoardOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if count := columnByState(all, "TODO").Count; count != 6 {
		t.Fatalf("empty board/sprint scopes changed default behavior: count=%d, want 6", count)
	}
}
