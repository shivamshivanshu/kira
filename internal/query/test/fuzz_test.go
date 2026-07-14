package query_test

import (
	"errors"
	"testing"

	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/query"
)

func fuzzOpts() query.Options {
	snap := id.Snapshot{
		Key: "KIRA",
		Items: []id.Item{
			{ULID: "01J8X8Q7RZTN5Y3VXW2A9K4E70", Number: "KIRA-1", Aliases: []string{"KIRA-8"}},
			{ULID: "01J8X8Q7RZTN5Y3VXW2A9K4E71", Number: "KIRA-2"},
			{ULID: "01J8X8Q7RZTN5Y3VXW2A9K4E72", Number: "KIRA-3"},
		},
	}
	return query.Options{
		Resolver:     id.NewResolver(snap),
		Priorities:   []string{"P0", "P1", "P2"},
		ActiveSprint: "2026-S14",
		Me:           "shivam",
	}
}

func FuzzCompile(f *testing.F) {
	seeds := []string{
		"", "bug", "\"quoted term\"",
		"state=TODO", "state!=DONE", "priority>=P1", "priority<P2", "estimate<3", "estimate>=1.5",
		"owner=@me", "reporter=@me", "label=infra", "sprint=active", "sprint=2026-S14",
		"due<2026-01-01", "created>=2026-01-01", "updated<=2026-12-31",
		"epic=KIRA-1", "blocked_by=KIRA-2", "links=KIRA-3",
		"type=ticket AND state=TODO", "state=TODO OR state=DONE", "NOT state=DONE",
		"(state=TODO OR state=REVIEW) AND owner=@me",
		"state IN (TODO, REVIEW, DONE)", "label IS EMPTY", "owner IS NOT EMPTY",
		"type=ticket ORDER BY priority DESC", "state=TODO ORDER BY updated asc",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	opts := fuzzOpts()
	f.Fuzz(func(t *testing.T, input string) {
		_, err := query.Compile(input, opts)
		if err == nil {
			return
		}
		var qerr *query.Error
		if errors.As(err, &qerr) && (qerr.Pos < 0 || qerr.Pos > len(input)) {
			t.Fatalf("error position %d out of bounds [0,%d] for input %q", qerr.Pos, len(input), input)
		}
	})
}
