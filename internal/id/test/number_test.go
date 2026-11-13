package id_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/shivamshivanshu/kira/internal/id"
)

func TestHighestNEmptyIsZero(t *testing.T) {
	t.Parallel()
	if got := id.HighestN(id.Snapshot{Key: "KIRA"}, "KIRA"); got != 0 {
		t.Fatalf("HighestN(empty) = %d, want 0", got)
	}
}

func TestHighestNIsMaxOfNumbersAndAliases(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(1))
	for trial := range 500 {
		var items []id.Item
		highest := 0
		nItems := rng.Intn(8)
		for range nItems {
			num := rng.Intn(1000) + 1
			if num > highest {
				highest = num
			}
			var aliases []string
			for range rng.Intn(4) {
				a := rng.Intn(1000) + 1
				if a > highest {
					highest = a
				}
				aliases = append(aliases, fmt.Sprintf("KIRA-%d", a))
			}
			items = append(items, id.Item{
				ULID:    id.Mint().String(),
				Number:  fmt.Sprintf("KIRA-%d", num),
				Aliases: aliases,
			})
		}
		if got := id.HighestN(id.Snapshot{Key: "KIRA", Items: items}, "KIRA"); got != highest {
			t.Fatalf("trial %d: HighestN = %d, want max %d", trial, got, highest)
		}
	}
}

func TestHighestNCountsAliasesAndIgnoresForeignKeys(t *testing.T) {
	t.Parallel()
	snap := id.Snapshot{Key: "KIRA", Items: []id.Item{
		{Number: "KIRA-3"},
		{Number: "kira-5"},
		{Number: "KIRA-2", Aliases: []string{"KIRA-40"}},
		{Number: "OTHER-9000"},
		{Number: "not-a-number"},
	}}
	if got := id.HighestN(snap, "KIRA"); got != 40 {
		t.Fatalf("HighestN = %d, want 40 (max alias)", got)
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

func TestAllocatorSequential(t *testing.T) {
	t.Parallel()
	snap := id.Snapshot{Items: []id.Item{{Number: "KIRA-6"}}}
	alloc := id.NewAllocator(snap, "KIRA")
	if got := alloc.Alloc(); got != "KIRA-7" {
		t.Fatalf("Alloc = %q, want KIRA-7", got)
	}
	if got := alloc.Alloc(); got != "KIRA-8" {
		t.Fatalf("second Alloc = %q, want KIRA-8", got)
	}
}

func TestKeyOf(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"KIRA-142": "KIRA",
		"A-B-7":    "A-B",
		"KIRA-":    "KIRA",
		"KIRA":     "",
		"-5":       "",
		"":         "",
	}
	for in, want := range cases {
		if got := id.KeyOf(in); got != want {
			t.Errorf("KeyOf(%q) = %q, want %q", in, got, want)
		}
	}
}
