package id_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/id"
)

func TestAllocateEmptyStartsAtOne(t *testing.T) {
	t.Parallel()
	if got := id.Allocate(id.Snapshot{Key: "KIRA"}); got.N != 1 || got.String() != "KIRA-1" {
		t.Fatalf("Allocate(empty) = %v, want KIRA-1", got)
	}
}

func TestAllocateIsMaxUnionPlusOne(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 500; trial++ {
		var items []id.Item
		union := map[int]bool{}
		max := 0
		nItems := rng.Intn(8)
		for i := 0; i < nItems; i++ {
			num := rng.Intn(1000) + 1
			union[num] = true
			if num > max {
				max = num
			}
			var aliases []string
			for j := 0; j < rng.Intn(4); j++ {
				a := rng.Intn(1000) + 1
				union[a] = true
				if a > max {
					max = a
				}
				aliases = append(aliases, fmt.Sprintf("KIRA-%d", a))
			}
			items = append(items, id.Item{
				ULID:    id.Mint().String(),
				Number:  fmt.Sprintf("KIRA-%d", num),
				Aliases: aliases,
			})
		}
		got := id.Allocate(id.Snapshot{Key: "KIRA", Items: items})
		want := max + 1
		if got.N != want {
			t.Fatalf("trial %d: Allocate.N = %d, want max+1 = %d", trial, got.N, want)
		}
		if union[got.N] {
			t.Fatalf("trial %d: allocated %d is already in the union", trial, got.N)
		}
	}
}

func TestAllocateCountsAliasesAndIgnoresForeignKeys(t *testing.T) {
	t.Parallel()
	snap := id.Snapshot{Key: "KIRA", Items: []id.Item{
		{Number: "KIRA-3"},
		{Number: "kira-5"},
		{Number: "KIRA-2", Aliases: []string{"KIRA-40"}},
		{Number: "OTHER-9000"},
		{Number: "not-a-number"},
	}}
	if got := id.Allocate(snap); got.N != 41 {
		t.Fatalf("Allocate.N = %d, want 41 (max alias 40 + 1)", got.N)
	}
}

func TestParseNumber(t *testing.T) {
	t.Parallel()
	ok := map[string]id.Number{
		"KIRA-142": {Key: "KIRA", N: 142},
		"kira-1":   {Key: "kira", N: 1},
		"A-B-7":    {Key: "A-B", N: 7},
	}
	for in, want := range ok {
		got, err := id.ParseNumber(in)
		if err != nil || got != want {
			t.Errorf("ParseNumber(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "KIRA", "KIRA-", "-5", "KIRA-0", "KIRA-x", "KIRA-1.5"} {
		if _, err := id.ParseNumber(in); err == nil {
			t.Errorf("ParseNumber(%q) succeeded, want error", in)
		}
	}
}

func TestHashNumberDerivesFromULID(t *testing.T) {
	t.Parallel()
	u := id.Mint()
	got := id.HashNumber("KIRA", u)
	if got != id.HashNumber("KIRA", u) {
		t.Fatal("HashNumber not deterministic")
	}
	s := u.String()
	want := "KIRA-" + s[len(s)-6:]
	if got != want {
		t.Fatalf("HashNumber = %q, want %q", got, want)
	}
	if !strings.HasPrefix(got, "KIRA-") || len(got) != len("KIRA-")+6 {
		t.Fatalf("HashNumber = %q, want KIRA- + 6 chars", got)
	}
}
