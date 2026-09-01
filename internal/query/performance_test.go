package query

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
)

func TestShortCircuitSkipsRightPredicate(t *testing.T) {
	// A known blocker needs cfg.CategoryOf. A nil config makes accidental
	// evaluation visible without adding a production instrumentation seam.
	it := &datamodel.Item{ID: "one", State: "TODO", BlockedBy: []string{"two"}}
	opts := Options{Items: []*datamodel.Item{it, {ID: "two", Type: "ticket", State: "TODO"}}}
	for _, expr := range []string{"state=DONE AND blocked", "state=TODO OR blocked"} {
		q, err := Compile(expr, opts)
		if err != nil {
			t.Fatal(err)
		}
		q.Pred(it, nil)
	}
	if _, err := Compile("state=TODO OR nonexistent=bad", opts); err == nil {
		t.Fatal("short circuit suppressed compilation error")
	}
}

func TestLargeScalarInMatchesAlternatives(t *testing.T) {
	values := []string{"one", "two", "three", "four", "five", "six", "seven", "@me"}
	for _, field := range []string{"owner", "board", "custom"} {
		calls := 0
		opts := Options{
			Schemas: map[string]entityschema.Schema{"ticket": {Name: "ticket",
				Fields: []entityschema.FieldDef{{Name: "custom", Type: entityschema.FieldString}}}},
			Me: func() string { calls++; return "Ada" },
		}
		q, err := Compile(field+" IN ("+strings.Join(values, ",")+")", opts)
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range append(values, "Ada", "missing", "") {
			it := &datamodel.Item{Type: "ticket", Owner: new(value), Number: value + "-1",
				ExtraScalars: map[string]string{"custom": value}}
			want := false
			for _, v := range values {
				pred, _, err := Match(field, v, opts)
				if err != nil {
					t.Fatal(err)
				}
				want = want || pred(it, nil)
			}
			if got := q.Pred(it, nil); got != want {
				t.Fatalf("%s IN for %q = %v, want %v", field, value, got, want)
			}
		}
		if field != "owner" && calls != 0 {
			t.Fatal("ordinary string @me resolved identity")
		}
	}
}
