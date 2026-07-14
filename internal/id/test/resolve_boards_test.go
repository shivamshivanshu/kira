package id_test

import (
	"errors"
	"testing"

	"github.com/shivamshivanshu/kira/internal/id"
)

func multiBoardSnapshot() id.Snapshot {
	return id.Snapshot{Key: "ABC", Items: []id.Item{
		{ULID: uA, Number: "ABC-1"},
		{ULID: uB, Number: "XYZ-1"},
		{ULID: u1, Number: "ABC-2"},
	}}
}

func TestBareNumberUniqueAcrossBoardsResolves(t *testing.T) {
	r := id.NewResolver(multiBoardSnapshot())
	got, err := r.Resolve("2")
	if err != nil {
		t.Fatalf("Resolve(2) error: %v", err)
	}
	if got != u1 {
		t.Fatalf("Resolve(2) = %q, want %q", got, u1)
	}
}

func TestBareNumberAmbiguousAcrossBoardsErrors(t *testing.T) {
	r := id.NewResolver(multiBoardSnapshot())
	_, err := r.Resolve("1")
	var amb *id.AmbiguousError
	if !errors.As(err, &amb) {
		t.Fatalf("Resolve(1) across boards must be ambiguous, got %v", err)
	}
	want := map[string]bool{"ABC-1": true, "XYZ-1": true}
	if len(amb.Candidates) != 2 {
		t.Fatalf("want 2 candidates, got %v", amb.Candidates)
	}
	for _, c := range amb.Candidates {
		if !want[c] {
			t.Fatalf("unexpected candidate %q in %v", c, amb.Candidates)
		}
	}
}

func TestFullNumberHeldAsAliasByTwoItemsIsAmbiguous(t *testing.T) {
	snap := id.Snapshot{Key: "KIRA", Items: []id.Item{
		{ULID: uA, Number: "KIRA-10", Aliases: []string{"KIRA-1"}},
		{ULID: uB, Number: "KIRA-11", Aliases: []string{"KIRA-1"}},
	}}
	r := id.NewResolver(snap)
	_, err := r.Resolve("KIRA-1")
	var amb *id.AmbiguousError
	if !errors.As(err, &amb) {
		t.Fatalf("KIRA-1 held as an alias by two items must be ambiguous, got %v", err)
	}
	want := map[string]bool{"KIRA-10": true, "KIRA-11": true}
	for _, c := range amb.Candidates {
		if !want[c] {
			t.Fatalf("unexpected candidate %q in %v", c, amb.Candidates)
		}
	}
}

func TestLiveNumberBeatsStaleAlias(t *testing.T) {
	snap := id.Snapshot{Key: "KIRA", Items: []id.Item{
		{ULID: uA, Number: "KIRA-1"},
		{ULID: uB, Number: "KIRA-2", Aliases: []string{"KIRA-1"}},
	}}
	r := id.NewResolver(snap)
	got, err := r.Resolve("KIRA-1")
	if err != nil || got != uA {
		t.Fatalf("live KIRA-1 must win over a stale alias; got %q err %v", got, err)
	}
}

func TestMoveAliasResolvesBareAndFull(t *testing.T) {
	snap := id.Snapshot{Key: "XYZ", Items: []id.Item{
		{ULID: uA, Number: "XYZ-7", Aliases: []string{"ABC-12"}},
		{ULID: uB, Number: "XYZ-1"},
	}}
	r := id.NewResolver(snap)
	for _, tok := range []string{"ABC-12", "abc-12", "12"} {
		got, err := r.Resolve(tok)
		if err != nil {
			t.Fatalf("Resolve(%q) error: %v", tok, err)
		}
		if got != uA {
			t.Fatalf("Resolve(%q) = %q, want %q", tok, got, uA)
		}
	}
}
