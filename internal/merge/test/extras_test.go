package merge_test

import (
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/merge"
)

func scalarExtras(kv ...string) map[string]datamodel.Rendered {
	out := make(map[string]datamodel.Rendered, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		out[kv[i]] = datamodel.Rendered(kv[i+1])
	}
	return out
}

func watchers(v ...string) map[string][]datamodel.Rendered {
	out := make([]datamodel.Rendered, len(v))
	for i, e := range v {
		out[i] = datamodel.Rendered(e)
	}
	return map[string][]datamodel.Rendered{"watchers": out}
}

func TestExtraSurvivesUnrelatedFieldMerge(t *testing.T) {
	t.Parallel()
	withExtra := func(it *datamodel.Item) { it.ExtraScalars = scalarExtras("team", "infra") }
	b := base(withExtra)
	ours := base(func(it *datamodel.Item) { withExtra(it); it.Title = "New title" })
	theirs := base(func(it *datamodel.Item) { withExtra(it); it.Owner = new("bob") })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	if got.ExtraScalars["team"] != "infra" {
		t.Fatalf("ExtraScalars[team] = %q, want infra to survive", got.ExtraScalars["team"])
	}
	if got.Title != "New title" || got.Owner == nil || *got.Owner != "bob" {
		t.Fatalf("builtin edits lost: title=%q owner=%v", got.Title, got.Owner)
	}
}

func TestExtraListConcurrentAppendsUnion(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.ExtraLists = watchers("alice") })
	ours := base(func(it *datamodel.Item) { it.ExtraLists = watchers("alice", "bob") })
	theirs := base(func(it *datamodel.Item) { it.ExtraLists = watchers("alice", "carol") })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	want := []datamodel.Rendered{"alice", "bob", "carol"}
	if !slices.Equal(got.ExtraLists["watchers"], want) {
		t.Fatalf("ExtraLists[watchers] = %v, want union %v", got.ExtraLists["watchers"], want)
	}
}

func TestExtrasDifferentKeysMergeWithoutArbitration(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.ExtraScalars = scalarExtras("k1", "x", "k2", "y") })
	ours := base(func(it *datamodel.Item) { it.ExtraScalars = scalarExtras("k1", "x2", "k2", "y") })
	theirs := base(func(it *datamodel.Item) { it.ExtraScalars = scalarExtras("k1", "x", "k2", "y2") })
	res := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger)
	if got := res.Item.ExtraScalars["k1"]; got != "x2" {
		t.Fatalf("ExtraScalars[k1] = %q, want ours' edit x2", got)
	}
	if got := res.Item.ExtraScalars["k2"]; got != "y2" {
		t.Fatalf("ExtraScalars[k2] = %q, want theirs' edit y2", got)
	}
	if len(res.Arbitrated) != 0 {
		t.Fatalf("Arbitrated = %v, want none for edits to different extras", res.Arbitrated)
	}
}

func TestExtraDeletedOnOneSideStaysDeleted(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.ExtraScalars = scalarExtras("team", "infra") })
	ours := base(nil)
	theirs := base(func(it *datamodel.Item) { it.ExtraScalars = scalarExtras("team", "infra") })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	if got.ExtraScalars != nil {
		t.Fatalf("ExtraScalars = %v, want one-sided deletion to hold", got.ExtraScalars)
	}
}

func TestExtraUntouchedEmptyListSurvives(t *testing.T) {
	t.Parallel()
	emptyTags := func(it *datamodel.Item) { it.ExtraLists = map[string][]datamodel.Rendered{"tags": {}} }
	b := base(emptyTags)
	ours := base(func(it *datamodel.Item) { emptyTags(it); it.Title = "New title" })
	theirs := base(func(it *datamodel.Item) { emptyTags(it); it.Owner = new("bob") })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	tags, ok := got.ExtraLists["tags"]
	if !ok || tags == nil || len(tags) != 0 {
		t.Fatalf("ExtraLists[tags] = %v present=%v, want the untouched empty list kept", tags, ok)
	}
}

func TestExtraListUnchangedKeptVerbatim(t *testing.T) {
	t.Parallel()
	withDupes := func(it *datamodel.Item) { it.ExtraLists = watchers("alice", "alice", "bob") }
	got := merge.Merge(base(withDupes), base(withDupes), base(withDupes), merge.Theirs, conflictMerger).Item
	if want := []datamodel.Rendered{"alice", "alice", "bob"}; !slices.Equal(got.ExtraLists["watchers"], want) {
		t.Fatalf("ExtraLists[watchers] = %v, want unchanged input back verbatim %v", got.ExtraLists["watchers"], want)
	}
}

func TestExtraListOneSidedReorderTaken(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.ExtraLists = watchers("alice", "bob") })
	ours := base(func(it *datamodel.Item) { it.ExtraLists = watchers("alice", "bob") })
	theirs := base(func(it *datamodel.Item) { it.ExtraLists = watchers("bob", "alice") })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	if want := []datamodel.Rendered{"bob", "alice"}; !slices.Equal(got.ExtraLists["watchers"], want) {
		t.Fatalf("ExtraLists[watchers] = %v, want theirs' reorder %v", got.ExtraLists["watchers"], want)
	}
}

func slaBlock(v datamodel.Rendered) func(*datamodel.Item) {
	return func(it *datamodel.Item) { it.ExtraBlocks = map[string]datamodel.Rendered{"sla": v} }
}

func TestExtraBlockChangedOnOneSideTakesThatSide(t *testing.T) {
	t.Parallel()
	b := base(slaBlock("response: 4h"))
	ours := base(slaBlock("response: 4h"))
	theirs := base(slaBlock("response: 1h\nresolve: 8h"))
	res := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger)
	if got := res.Item.ExtraBlocks["sla"]; got != "response: 1h\nresolve: 8h" {
		t.Fatalf("ExtraBlocks[sla] = %q, want theirs' edit", got)
	}
	if len(res.Arbitrated) != 0 {
		t.Fatalf("Arbitrated = %v, want none for a one-sided block edit", res.Arbitrated)
	}
}

// kira does not read inside a block, so it cannot merge inside one: the whole
// value goes to the later-updated side and the key is reported as arbitrated.
func TestExtraBlockChangedOnBothSidesArbitratesAsAUnit(t *testing.T) {
	t.Parallel()
	b := base(slaBlock("response: 4h"))
	ours := base(func(it *datamodel.Item) {
		slaBlock("response: 1h")(it)
		it.Updated = "2026-02-02T00:00:00Z"
	})
	theirs := base(func(it *datamodel.Item) {
		slaBlock("response: 4h\nresolve: 8h")(it)
		it.Updated = "2026-02-01T00:00:00Z"
	})
	res := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger)
	if got := res.Item.ExtraBlocks["sla"]; got != "response: 1h" {
		t.Fatalf("ExtraBlocks[sla] = %q, want ours whole (updated later); no partial merge", got)
	}
	if !slices.Contains(res.Arbitrated, "sla") {
		t.Fatalf("Arbitrated = %v, want sla reported", res.Arbitrated)
	}
}

func TestExtraKeyFlippingBetweenScalarAndBlock(t *testing.T) {
	t.Parallel()
	b := base(nil)
	ours := base(func(it *datamodel.Item) {
		it.ExtraScalars = scalarExtras("sla", "4h")
		it.Updated = "2026-02-01T00:00:00Z"
	})
	theirs := base(func(it *datamodel.Item) {
		slaBlock("response: 4h")(it)
		it.Updated = "2026-02-02T00:00:00Z"
	})
	got := merge.Merge(b, ours, theirs, merge.Ours, conflictMerger).Item
	if _, isScalar := got.ExtraScalars["sla"]; isScalar {
		t.Fatalf("sla must not survive in two maps: scalars=%v blocks=%v", got.ExtraScalars, got.ExtraBlocks)
	}
	if want := datamodel.Rendered("response: 4h"); got.ExtraBlocks["sla"] != want {
		t.Fatalf("ExtraBlocks[sla] = %q, want %q (theirs updated later, wins wholesale)", got.ExtraBlocks["sla"], want)
	}
	if _, err := codec.Parse(codec.Serialize(got)); err != nil {
		t.Fatalf("merged item must re-parse: %v", err)
	}
}
