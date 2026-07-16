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
		highest := 0
		nItems := rng.Intn(8)
		for i := 0; i < nItems; i++ {
			num := rng.Intn(1000) + 1
			union[num] = true
			if num > highest {
				highest = num
			}
			var aliases []string
			for j := 0; j < rng.Intn(4); j++ {
				a := rng.Intn(1000) + 1
				union[a] = true
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
		got := id.Allocate(id.Snapshot{Key: "KIRA", Items: items})
		want := highest + 1
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

func mustULID(t *testing.T, s string) id.ULID {
	t.Helper()
	u, err := id.ParseULID(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func hashAlloc(snap id.Snapshot) *id.Allocator { return id.NewAllocator(true, snap, "KIRA") }

func TestAllocatorHashDerivesFromULID(t *testing.T) {
	t.Parallel()
	u := mustULID(t, "01AN4Z07BY79KA1307SR9X4MV3")
	got := hashAlloc(id.Snapshot{}).Alloc(u)
	if got != hashAlloc(id.Snapshot{}).Alloc(u) {
		t.Fatal("hash allocation not deterministic for a fresh snapshot")
	}
	s := u.String()
	want := "KIRA-" + s[len(s)-6:]
	if got != want {
		t.Fatalf("Alloc = %q, want %q", got, want)
	}
	if !strings.HasPrefix(got, "KIRA-") || len(got) != len("KIRA-")+6 {
		t.Fatalf("Alloc = %q, want KIRA- + 6 chars", got)
	}
}

func TestAllocatorHashRetriesOnSnapshotCollision(t *testing.T) {
	t.Parallel()
	u := mustULID(t, "01AN4Z07BY79KA1307SR9X4MV3")
	s := u.String()
	snap := id.Snapshot{Items: []id.Item{{Number: "KIRA-" + s[len(s)-6:]}}}
	got := hashAlloc(snap).Alloc(u)
	if want := "KIRA-" + s[len(s)-7:]; got != want {
		t.Fatalf("Alloc on collision = %q, want widened %q", got, want)
	}
	snap.Items = append(snap.Items, id.Item{Number: "OTHER-1", Aliases: []string{got}})
	if got := hashAlloc(snap).Alloc(u); got != "KIRA-"+s[len(s)-8:] {
		t.Fatalf("Alloc on alias collision = %q, want %q", got, "KIRA-"+s[len(s)-8:])
	}
}

func TestAllocatorHashRemembersEarlierAllocations(t *testing.T) {
	t.Parallel()
	a := mustULID(t, "01AN4Z07BY79KA1307SR9X4MV3")
	b := mustULID(t, "01BN4Z07BY79KA1307SR9X4MV3")
	alloc := hashAlloc(id.Snapshot{})
	first, second := alloc.Alloc(a), alloc.Alloc(b)
	if first != "KIRA-9X4MV3" || second != "KIRA-R9X4MV3" {
		t.Fatalf("colliding suffixes allocated %q then %q, want KIRA-9X4MV3 then KIRA-R9X4MV3", first, second)
	}
	r := id.NewResolver(id.Snapshot{Key: "KIRA", Items: []id.Item{
		{ULID: a.String(), Number: first},
		{ULID: b.String(), Number: second},
	}})
	for number, want := range map[string]string{first: a.String(), second: b.String()} {
		got, err := r.Resolve(number)
		if err != nil || got != want {
			t.Fatalf("Resolve(%q) = %q, %v; want %q", number, got, err, want)
		}
	}
}

func TestAllocatorHashSkipsAllDigitSuffix(t *testing.T) {
	t.Parallel()
	u := mustULID(t, "01AN4Z07BY79KA1307SR000123")
	got := hashAlloc(id.Snapshot{}).Alloc(u)
	if got != "KIRA-R000123" {
		t.Fatalf("Alloc = %q, want KIRA-R000123 (all-digit suffix widened)", got)
	}
	if _, err := id.ParseNumber(got); err == nil {
		t.Fatalf("hash number %q must not parse as a sequential KEY-n number", got)
	}
}

func TestAllocatorSequential(t *testing.T) {
	t.Parallel()
	snap := id.Snapshot{Items: []id.Item{{Number: "KIRA-6"}}}
	alloc := id.NewAllocator(false, snap, "KIRA")
	if got := alloc.Alloc(id.Mint()); got != "KIRA-7" {
		t.Fatalf("Alloc = %q, want KIRA-7", got)
	}
	if got := alloc.Alloc(id.Mint()); got != "KIRA-8" {
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
