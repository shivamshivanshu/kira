package reconcile_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/reconcile"
)

func keyOf(number string) string {
	i := strings.LastIndexByte(number, '-')
	return number[:i]
}

func TestPlanRenumbersOnOwnBoard(t *testing.T) {
	s := snap(
		id.Item{ULID: "A", Number: "ABC-1"},
		id.Item{ULID: "B", Number: "ABC-1"},
		id.Item{ULID: "C", Number: "XYZ-1"},
		id.Item{ULID: "D", Number: "XYZ-1"},
	)
	for _, r := range reconcile.Plan(s) {
		if keyOf(r.From) != keyOf(r.To) {
			t.Fatalf("renumber crossed boards: %s -> %s", r.From, r.To)
		}
	}
}

func TestPlanMultiBoardConverges(t *testing.T) {
	keys := []string{"ABC", "XYZ", "KIRA"}
	for seed := int64(0); seed < 300; seed++ {
		rng := rand.New(rand.NewSource(seed))
		n := rng.Intn(18) + 2
		items := make([]id.Item, n)
		for i := 0; i < n; i++ {
			key := keys[rng.Intn(len(keys))]
			num := rng.Intn(5) + 1
			var aliases []string
			if rng.Intn(3) == 0 {
				aliases = []string{fmt.Sprintf("%s-%d", keys[rng.Intn(len(keys))], rng.Intn(5)+1)}
			}
			items[i] = id.Item{ULID: fmt.Sprintf("U%04d", i), Number: fmt.Sprintf("%s-%d", key, num), Aliases: aliases}
		}
		s := snap(items...)
		plan := reconcile.Plan(s)

		if !reflect.DeepEqual(plan, reconcile.Plan(s)) {
			t.Fatalf("seed %d: non-deterministic plan", seed)
		}

		counts := map[string]int{}
		minHolder := map[string]string{}
		for _, it := range items {
			counts[it.Number]++
			if cur, ok := minHolder[it.Number]; !ok || it.ULID < cur {
				minHolder[it.Number] = it.ULID
			}
		}
		wantLen := 0
		for _, c := range counts {
			if c > 1 {
				wantLen += c - 1
			}
		}
		if len(plan) != wantLen {
			t.Fatalf("seed %d: plan not minimal: len %d, want %d", seed, len(plan), wantLen)
		}
		for _, r := range plan {
			if r.ULID == minHolder[r.From] {
				t.Fatalf("seed %d: min-ULID holder %s of %s was renumbered", seed, r.ULID, r.From)
			}
		}

		applied := make([]id.Item, n)
		byULID := map[string]int{}
		for i := range items {
			applied[i] = items[i]
			byULID[items[i].ULID] = i
		}
		for _, r := range plan {
			if keyOf(r.From) != keyOf(r.To) {
				t.Fatalf("seed %d: renumber crossed boards: %s -> %s", seed, r.From, r.To)
			}
			i := byULID[r.ULID]
			applied[i].Aliases = append(append([]string{}, applied[i].Aliases...), r.From)
			applied[i].Number = r.To
		}

		if p2 := reconcile.Plan(snap(applied...)); len(p2) != 0 {
			t.Fatalf("seed %d: did not converge, residual collisions: %v", seed, p2)
		}

		live := map[string]string{}
		for _, it := range applied {
			if prev, ok := live[it.Number]; ok {
				t.Fatalf("seed %d: duplicate live number %s on %s and %s", seed, it.Number, prev, it.ULID)
			}
			live[it.Number] = it.ULID
		}
	}
}
