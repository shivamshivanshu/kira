package id

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

const (
	u1 = "01AN4Z07BY79KA1307SR9X4MV3"
	uA = "01BX5ZAAAAAAAAAAAAAAAAAAAA"
	uB = "01BX5ZBBBBBBBBBBBBBBBBBBBB"
)

func testSnapshot() Snapshot {
	return Snapshot{Key: "KIRA", Items: []Item{
		{ULID: u1, Number: "KIRA-1"},
		{ULID: uA, Number: "KIRA-2", Aliases: []string{"KIRA-99"}},
		{ULID: uB, Number: "KIRA-3"},
	}}
}

func TestResolve(t *testing.T) {
	r := NewResolver(testSnapshot())
	cases := []struct {
		name  string
		token string
		want  string
	}{
		{"full ULID", u1, u1},
		{"full ULID lowercase", strings.ToLower(u1), u1},
		{"unique prefix", "01AN4Z", u1},
		{"unique prefix lowercase", "01bx5za", uA},
		{"number with key", "KIRA-2", uA},
		{"number case-insensitive key", "kira-2", uA},
		{"bare number", "3", uB},
		{"alias", "KIRA-99", uA},
		{"alias case-insensitive", "kira-99", uA},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := r.Resolve(c.token)
			if err != nil {
				t.Fatalf("Resolve(%q) error: %v", c.token, err)
			}
			if got != c.want {
				t.Fatalf("Resolve(%q) = %q, want %q", c.token, got, c.want)
			}
		})
	}
}

func TestResolveAmbiguousPrefixListsCandidates(t *testing.T) {
	r := NewResolver(testSnapshot())
	_, err := r.Resolve("01BX5Z")
	var amb *AmbiguousError
	if !errors.As(err, &amb) {
		t.Fatalf("Resolve(ambiguous) err = %v, want *AmbiguousError", err)
	}
	if want := []string{uA, uB}; !reflect.DeepEqual(amb.Candidates, want) {
		t.Fatalf("candidates = %v, want %v", amb.Candidates, want)
	}
}

func TestResolvePrefixPrecedesBareNumber(t *testing.T) {
	// "01" is a prefix of all three ULIDs; the fixed order tries prefix before
	// bare number, so this is ambiguous rather than resolving KIRA-01.
	r := NewResolver(testSnapshot())
	_, err := r.Resolve("01")
	var amb *AmbiguousError
	if !errors.As(err, &amb) {
		t.Fatalf("Resolve(\"01\") err = %v, want *AmbiguousError", err)
	}
	if len(amb.Candidates) != 3 {
		t.Fatalf("candidates = %v, want all 3 ULIDs", amb.Candidates)
	}
}

func TestResolveLiveNumberBeatsAlias(t *testing.T) {
	// uA holds KIRA-5 as a live number; uB retired it into aliases. Resolving
	// KIRA-5 must reach the live holder, not the alias holder.
	r := NewResolver(Snapshot{Key: "KIRA", Items: []Item{
		{ULID: uB, Number: "KIRA-3", Aliases: []string{"KIRA-5"}},
		{ULID: uA, Number: "KIRA-5"},
	}})
	got, err := r.Resolve("KIRA-5")
	if err != nil || got != uA {
		t.Fatalf("Resolve(KIRA-5) = %q, %v; want %q (live number wins)", got, err, uA)
	}
}

func TestResolveNotFound(t *testing.T) {
	r := NewResolver(testSnapshot())
	for _, token := range []string{
		"",
		"KIRA-999",
		"9",
		"01AN4Z07BY79KA1307SR9X4MV4", // valid full ULID, not in snapshot
		"garbage-token",
	} {
		_, err := r.Resolve(token)
		var nf *NotFoundError
		if !errors.As(err, &nf) {
			t.Errorf("Resolve(%q) err = %v, want *NotFoundError", token, err)
		}
	}
}
