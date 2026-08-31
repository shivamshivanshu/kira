package query_test

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/query"
)

func extrasFixture() (items []*datamodel.Item, opts query.Options, cfg *datamodel.Config) {
	items, opts, cfg = evalFixture()
	opts.Extras = []query.ExtraField{
		{Name: "team"},
		{Name: "watchers", List: true},
		{Name: "points", Ordered: true},
	}
	items[1].Extras = map[string]datamodel.ExtraValue{
		"team":     {Scalar: new("'infra core'")},
		"watchers": {List: []string{"alice", "'b c'"}},
		"points":   {Scalar: new("8")},
	}
	items[2].Extras = map[string]datamodel.ExtraValue{
		"team":     {Scalar: new("qa")},
		"watchers": {List: []string{}},
		"points":   {Scalar: new("10")},
	}
	return items, opts, cfg
}

func TestEvalExtras(t *testing.T) {
	items, opts, cfg := extrasFixture()
	tests := []struct {
		expr string
		want []string
	}{
		{"team=qa", []string{"KIRA-2"}},
		{`team="infra core"`, []string{"KIRA-1"}},
		{"team!=qa", []string{"KIRA-1", "KIRA-100", "KIRA-3"}},
		{"team IN (qa, sre)", []string{"KIRA-2"}},
		{"team IS EMPTY", []string{"KIRA-100", "KIRA-3"}},
		{"team IS NOT EMPTY", []string{"KIRA-1", "KIRA-2"}},
		{"watchers=alice", []string{"KIRA-1"}},
		{`watchers="b c"`, []string{"KIRA-1"}},
		{"watchers!=alice", []string{"KIRA-100", "KIRA-2", "KIRA-3"}},
		{"watchers IS EMPTY", []string{"KIRA-100", "KIRA-2", "KIRA-3"}},
		{"watchers IS NOT EMPTY", []string{"KIRA-1"}},
		{"points=8", []string{"KIRA-1"}},
		{"points!=8", []string{"KIRA-100", "KIRA-2", "KIRA-3"}},
		{"points>8", []string{"KIRA-2"}},
		{"points>=8", []string{"KIRA-1", "KIRA-2"}},
		{"points<8", nil},
		{"points IS EMPTY", []string{"KIRA-100", "KIRA-3"}},
		{"team=qa AND points>8", []string{"KIRA-2"}},
	}
	for _, tc := range tests {
		got := matchNums(t, tc.expr, items, opts, cfg)
		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Errorf("%q matched %v, want %v", tc.expr, got, tc.want)
		}
	}
}

func TestOrderByExtraIsNumericWhenOrdered(t *testing.T) {
	items, opts, cfg := extrasFixture()
	q, err := query.Parse("type=ticket ORDER BY points", opts.Extras)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	keyOf := q.Order.Keyer(cfg)
	if q.Order.Less(keyOf(items[2]), keyOf(items[1])) {
		t.Error("points must order numerically: 10 sorted before 8")
	}
	if !q.Order.Less(keyOf(items[1]), keyOf(items[3])) {
		t.Error("an item without the extra must sort after one with it")
	}
}

func TestExtrasFieldErrors(t *testing.T) {
	_, opts, _ := extrasFixture()
	cases := []struct {
		name, expr string
		unknown    bool
		wantMsg    string
	}{
		{"unknown field lists extras", "bogus = x", true, "team"},
		{"extras unknown without spec", "team = x", true, "unknown field"},
		{"syntax error is not unknown-field", "state =", false, "expected a value"},
		{"ordered extra needs a number", "points > abc", false, "invalid number"},
		{"no ordered cmp on plain extra", "team > qa", false, "only valid on"},
		{"no order by list extra", "x ORDER BY watchers", false, "cannot order by list field"},
	}
	for _, tc := range cases {
		extras := opts.Extras
		if tc.name == "extras unknown without spec" {
			extras = nil
		}
		_, err := query.Compile(tc.expr, query.Options{Extras: extras})
		if err == nil {
			t.Errorf("%s: Compile(%q) should error", tc.name, tc.expr)
			continue
		}
		if !strings.Contains(err.Error(), tc.wantMsg) {
			t.Errorf("%s: Compile(%q) = %v, want message containing %q", tc.name, tc.expr, err, tc.wantMsg)
		}
		if query.IsUnknownField(err) != tc.unknown {
			t.Errorf("%s: IsUnknownField = %v, want %v", tc.name, !tc.unknown, tc.unknown)
		}
	}
}

func TestMatchOnExtras(t *testing.T) {
	items, opts, cfg := extrasFixture()
	pred, _, err := query.Match("team", "qa", opts)
	if err != nil {
		t.Fatalf("Match(team): %v", err)
	}
	if !pred(items[2], cfg) || pred(items[1], cfg) {
		t.Error("Match(team, qa) must match KIRA-2 only")
	}
	pred, _, err = query.Match("watchers", "alice", opts)
	if err != nil {
		t.Fatalf("Match(watchers): %v", err)
	}
	if !pred(items[1], cfg) || pred(items[2], cfg) {
		t.Error("Match(watchers, alice) must match KIRA-1 only")
	}
	if _, _, err := query.Match("team", "qa", query.Options{}); err == nil {
		t.Error("Match on an extra without the spec must error")
	}
}
