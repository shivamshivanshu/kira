package reconcile_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/reconcile"
)

func snap(items ...id.Item) id.Snapshot {
	return id.Snapshot{Key: "KIRA", Items: items}
}

func TestNoCollisionNoPlan(t *testing.T) {
	s := snap(
		id.Item{ULID: "A", Number: "KIRA-1"},
		id.Item{ULID: "B", Number: "KIRA-2"},
	)
	if p := reconcile.Plan(s); len(p) != 0 {
		t.Fatalf("expected empty plan, got %v", p)
	}
}

func TestLiveCollisionRenumbersLaterULID(t *testing.T) {
	s := snap(
		id.Item{ULID: "A", Number: "KIRA-1"},
		id.Item{ULID: "B", Number: "KIRA-1"},
	)
	p := reconcile.Plan(s)
	if len(p) != 1 {
		t.Fatalf("expected 1 renumber, got %d: %v", len(p), p)
	}
	if p[0].ULID != "B" || p[0].From != "KIRA-1" || p[0].To != "KIRA-2" {
		t.Fatalf("wrong renumber: %+v (want B KIRA-1 -> KIRA-2)", p[0])
	}
}

func TestNextFreeSkipsAliasReservedNumbers(t *testing.T) {
	s := snap(
		id.Item{ULID: "A", Number: "KIRA-1"},
		id.Item{ULID: "B", Number: "KIRA-1"},
		id.Item{ULID: "C", Number: "KIRA-9", Aliases: []string{"KIRA-2"}},
	)
	p := reconcile.Plan(s)
	if len(p) != 1 || p[0].To != "KIRA-10" {
		t.Fatalf("next-free must skip alias-reserved KIRA-2 and live KIRA-9; got %v", p)
	}
}

func TestThreeWayCollisionKeepsEarliestRenumbersRest(t *testing.T) {
	s := snap(
		id.Item{ULID: "C", Number: "KIRA-5"},
		id.Item{ULID: "A", Number: "KIRA-5"},
		id.Item{ULID: "B", Number: "KIRA-5"},
	)
	p := reconcile.Plan(s)
	if len(p) != 2 {
		t.Fatalf("expected 2 renumbers, got %v", p)
	}
	if p[0].ULID != "B" || p[1].ULID != "C" {
		t.Fatalf("earliest ULID A must be kept; got renumbers on %s,%s", p[0].ULID, p[1].ULID)
	}
	if p[0].To == p[1].To {
		t.Fatalf("renumbers must be distinct, both %s", p[0].To)
	}
}

func TestDeterministic(t *testing.T) {
	s := snap(
		id.Item{ULID: "B", Number: "KIRA-1"},
		id.Item{ULID: "A", Number: "KIRA-1"},
		id.Item{ULID: "C", Number: "KIRA-2"},
		id.Item{ULID: "D", Number: "KIRA-2"},
	)
	first := reconcile.Plan(s)
	for i := 0; i < 50; i++ {
		if got := reconcile.Plan(s); !reflect.DeepEqual(got, first) {
			t.Fatalf("Plan not deterministic:\n%v\n%v", first, got)
		}
	}
}

func FuzzPlanInvariants(f *testing.F) {
	f.Add(uint16(0x1234), uint8(5))
	f.Add(uint16(0xFFFF), uint8(12))
	f.Fuzz(func(t *testing.T, seed uint16, n uint8) {
		items := genItems(seed, int(n%16))
		s := snap(items...)
		before := domain(items)
		plan := reconcile.Plan(s)

		applied := applyPlan(items, plan)

		// No item lost, ULIDs untouched.
		if len(applied) != len(items) {
			t.Fatalf("item count changed: %d -> %d", len(items), len(applied))
		}
		for i := range items {
			if applied[i].ULID != items[i].ULID {
				t.Fatalf("ULID rewritten: %q -> %q", items[i].ULID, applied[i].ULID)
			}
		}
		// Post-plan there is no live-number collision.
		seen := map[string]string{}
		for _, it := range applied {
			if other, dup := seen[it.Number]; dup {
				t.Fatalf("live collision remains on %s (%s, %s)", it.Number, other, it.ULID)
			}
			seen[it.Number] = it.ULID
		}
		// Every new number is fresh (outside the original domain) and distinct.
		fresh := map[string]bool{}
		for _, r := range plan {
			if before[r.To] {
				t.Fatalf("reissued number %s already in domain", r.To)
			}
			if fresh[r.To] {
				t.Fatalf("duplicate new number %s", r.To)
			}
			fresh[r.To] = true
			// Retired number is preserved as an alias.
			it := applied[indexOf(applied, r.ULID)]
			if !contains(it.Aliases, r.From) {
				t.Fatalf("retired number %s not appended to aliases of %s", r.From, r.ULID)
			}
		}
	})
}

func genItems(seed uint16, n int) []id.Item {
	items := make([]id.Item, n)
	for i := 0; i < n; i++ {
		// Deliberately collide numbers so the fuzzer exercises repair.
		num := int((seed>>uint(i%8))%uint16(max(n, 1))) + 1
		items[i] = id.Item{
			ULID:   fmt.Sprintf("ULID%03d", i),
			Number: fmt.Sprintf("KIRA-%d", num),
		}
	}
	return items
}

func applyPlan(items []id.Item, plan []reconcile.Renumber) []id.Item {
	out := make([]id.Item, len(items))
	copy(out, items)
	for _, r := range plan {
		i := indexOf(out, r.ULID)
		out[i].Number = r.To
		out[i].Aliases = append(append([]string(nil), out[i].Aliases...), r.From)
	}
	return out
}

func domain(items []id.Item) map[string]bool {
	d := map[string]bool{}
	for _, it := range items {
		d[it.Number] = true
		for _, a := range it.Aliases {
			d[a] = true
		}
	}
	return d
}

func indexOf(items []id.Item, ulid string) int {
	for i, it := range items {
		if it.ULID == ulid {
			return i
		}
	}
	return -1
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
