package query_test

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/query"
)

func extrasFixture() (items []*datamodel.Item, opts query.Options, cfg *datamodel.Config) {
	items, opts, cfg = evalFixture()
	opts.Schemas = map[string]entityschema.Schema{
		datamodel.TypeTicket: {Name: datamodel.TypeTicket, Fields: []entityschema.FieldDef{
			{Name: "team", Type: entityschema.FieldString},
			{Name: "watchers", Type: entityschema.FieldString, List: true},
			{Name: "points", Type: entityschema.FieldInt},
		}},
	}
	items[1].ExtraScalars = map[string]datamodel.Rendered{"team": "'infra core'", "points": "8"}
	items[1].ExtraLists = map[string][]datamodel.Rendered{"watchers": {"alice", "'b c'"}}
	items[2].ExtraScalars = map[string]datamodel.Rendered{"team": "qa", "points": "10"}
	items[2].ExtraLists = map[string][]datamodel.Rendered{"watchers": {}}
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

// An extra reaches the query as the rendered YAML of its frontmatter node, so
// a value that needed quoting on disk only equals the query's bare text once
// it is decoded back. Every comparison path must decode; presence checks need
// not, since quoting cannot make a value absent.
func TestQuotedExtraValueDecodesBeforeMatching(t *testing.T) {
	items, opts, cfg := extrasFixture()
	if raw := items[1].ExtraScalars["team"]; raw != "'infra core'" {
		t.Fatalf("fixture must store the quoted rendering, got %q", raw)
	}
	// Eight alternatives take the set-based IN path, which extracts the field
	// once per item rather than once per alternative.
	exprs := []string{
		`team="infra core"`,
		`watchers="b c"`,
		`team IN ("infra core", a, b, c, d, e, f, g)`,
	}
	for _, expr := range exprs {
		if got := matchNums(t, expr, items, opts, cfg); strings.Join(got, ",") != "KIRA-1" {
			t.Errorf("%s matched %v, want KIRA-1", expr, got)
		}
	}
}

// An extra is declared by the schemas of some item types and not others, so a
// query spanning types matches only where it is declared.
func TestExtraUndeclaredForItemType(t *testing.T) {
	items, opts, cfg := extrasFixture()
	items[0].ExtraScalars = map[string]datamodel.Rendered{"team": "qa"}
	if got := matchNums(t, "team=qa", items, opts, cfg); strings.Join(got, ",") != "KIRA-2" {
		t.Errorf("team=qa matched %v, want only the ticket that declares team", got)
	}
}

func TestOrderByExtraIsNumericWhenOrdered(t *testing.T) {
	items, opts, cfg := extrasFixture()
	q, err := query.Compile("type=ticket ORDER BY points", opts)
	if err != nil {
		t.Fatalf("Compile: %v", err)
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
		{"extras unknown without schemas", "team = x", true, "unknown field"},
		{"ORDER BY unknown field", "x ORDER BY bogus", true, "unknown field"},
		{"syntax error is not unknown-field", "state =", false, "expected a value"},
		{"ordered extra needs a number", "points > abc", false, "invalid number"},
		{"no ordered cmp on plain extra", "team > qa", false, "declared string"},
		{"no order by list extra", "x ORDER BY watchers", false, "cannot order by list field"},
	}
	for _, tc := range cases {
		schemas := opts.Schemas
		if tc.name == "extras unknown without schemas" {
			schemas = nil
		}
		_, err := query.Compile(tc.expr, query.Options{Schemas: schemas})
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
		t.Error("Match on an extra without the schemas must error")
	}
}

func TestFieldNamesIncludesSchemaFields(t *testing.T) {
	_, opts, _ := extrasFixture()
	names := strings.Join(query.FieldNames(opts.Schemas), ",")
	for _, want := range []string{"state", "team", "watchers", "points"} {
		if !strings.Contains(names, want) {
			t.Errorf("FieldNames = %s, want it to include %q", names, want)
		}
	}
}
