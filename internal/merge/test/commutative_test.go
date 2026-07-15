package merge_test

import (
	"math/rand"
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/merge"
	"github.com/shivamshivanshu/kira/internal/ptr"
)

func swapSide(s merge.Side) merge.Side {
	if s == merge.Ours {
		return merge.Theirs
	}
	return merge.Ours
}

func pickStr(r *rand.Rand, pool []string) string { return pool[r.Intn(len(pool))] }

func pickPtr(r *rand.Rand, pool []string) *string {
	i := r.Intn(len(pool) + 1)
	if i == len(pool) {
		return nil
	}
	return ptr.To(pool[i])
}

func pickSubset(r *rand.Rand, pool []string) []string {
	var out []string
	for _, e := range pool {
		if r.Intn(2) == 0 {
			out = append(out, e)
		}
	}
	return out
}

func randLinks(r *rand.Rand) map[string][]string {
	var out map[string][]string
	for _, typ := range datamodel.LinkTypes {
		targets := pickSubset(r, []string{"K1", "K2"})
		if len(targets) == 0 {
			continue
		}
		if out == nil {
			out = map[string][]string{}
		}
		out[string(typ)] = targets
	}
	return out
}

func randSide(r *rand.Rand) *datamodel.Item {
	it := base(nil)
	it.Title = pickStr(r, []string{"Base title", "A title", "B title"})
	it.State = pickStr(r, []string{"TODO", "IN_PROGRESS", "REVIEW", "DONE"})
	it.Subtype = pickPtr(r, []string{"bug", "task"})
	it.Resolution = pickPtr(r, []string{"fixed", "dropped"})
	it.Priority = pickPtr(r, []string{"P1", "P2"})
	it.Rank = pickPtr(r, []string{"0|a:", "0|b:"})
	it.Owner = pickPtr(r, []string{"alice", "bob"})
	it.Reporter = pickPtr(r, []string{"carol", "dave"})
	it.Epic = pickPtr(r, []string{"01EPIC0000000000000000000A", "01EPIC0000000000000000000B"})
	it.Sprint = pickPtr(r, []string{"2026-S1", "2026-S2"})
	it.Due = pickPtr(r, []string{"2026-01-01", "2026-02-02"})
	if e := r.Intn(4); e < 3 {
		v := float64(e)
		it.Estimate = &v
	}
	it.Labels = pickSubset(r, []string{"a", "b", "c", "d"})
	it.BlockedBy = pickSubset(r, []string{"X", "Y", "Z"})
	it.Aliases = pickSubset(r, []string{"KIRA-90", "KIRA-91"})
	it.Links = randLinks(r)
	it.Body = pickStr(r, []string{"## Description\n\nbody\n", "## Description\n\nours prose\n", "## Description\n\ntheirs prose\n"})
	it.Updated = pickStr(r, []string{"2026-02-01T00:00:00Z", "2026-02-02T00:00:00Z", "2026-02-03T00:00:00Z"})
	return it
}

func sortedCopy(xs []string) []string {
	out := slices.Clone(xs)
	slices.Sort(out)
	return out
}

func canonItem(it *datamodel.Item) *datamodel.Item {
	c := *it
	c.Labels = sortedCopy(it.Labels)
	c.BlockedBy = sortedCopy(it.BlockedBy)
	c.Aliases = sortedCopy(it.Aliases)
	if it.Links != nil {
		links := make(map[string][]string, len(it.Links))
		for k, v := range it.Links {
			links[k] = sortedCopy(v)
		}
		c.Links = links
	}
	return &c
}

func TestMergeCommutativeUnderSideSwap(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(20260714))
	for i := 0; i < 1000; i++ {
		b := base(nil)
		a, c := randSide(r), randSide(r)
		remote := merge.Ours
		if r.Intn(2) == 0 {
			remote = merge.Theirs
		}
		fwd := merge.Merge(b, a, c, remote, conflictMerger)
		rev := merge.Merge(b, c, a, swapSide(remote), conflictMerger)

		if gotF, gotR := codec.Serialize(canonItem(fwd.Item)), codec.Serialize(canonItem(rev.Item)); gotF != gotR {
			t.Fatalf("iter %d: merge not symmetric under (ours<->theirs, remote swap)\n--- forward ---\n%s\n--- reverse ---\n%s", i, gotF, gotR)
		}
		if af, ar := sortedCopy(fwd.Arbitrated), sortedCopy(rev.Arbitrated); !slices.Equal(af, ar) {
			t.Fatalf("iter %d: arbitrated sets differ: %v vs %v", i, af, ar)
		}
	}
}
