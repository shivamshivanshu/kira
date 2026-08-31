package merge_test

import (
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/merge"
)

func scalarExtras(kv ...string) map[string]datamodel.ExtraValue {
	out := make(map[string]datamodel.ExtraValue, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		out[kv[i]] = datamodel.ExtraValue{Scalar: new(kv[i+1])}
	}
	return out
}

func watchers(vals ...string) map[string]datamodel.ExtraValue {
	return map[string]datamodel.ExtraValue{"watchers": {List: vals}}
}

func TestExtraSurvivesUnrelatedFieldMerge(t *testing.T) {
	t.Parallel()
	withExtra := func(it *datamodel.Item) { it.Extras = scalarExtras("team", "infra") }
	b := base(withExtra)
	ours := base(func(it *datamodel.Item) { withExtra(it); it.Title = "New title" })
	theirs := base(func(it *datamodel.Item) { withExtra(it); it.Owner = new("bob") })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	if v := got.Extras["team"]; v.Scalar == nil || *v.Scalar != "infra" {
		t.Fatalf("Extras[team] = %+v, want scalar infra to survive", v)
	}
	if got.Title != "New title" || got.Owner == nil || *got.Owner != "bob" {
		t.Fatalf("builtin edits lost: title=%q owner=%v", got.Title, got.Owner)
	}
}

func TestExtraListConcurrentAppendsUnion(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.Extras = watchers("alice") })
	ours := base(func(it *datamodel.Item) { it.Extras = watchers("alice", "bob") })
	theirs := base(func(it *datamodel.Item) { it.Extras = watchers("alice", "carol") })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	want := []string{"alice", "bob", "carol"}
	if !slices.Equal(got.Extras["watchers"].List, want) {
		t.Fatalf("Extras[watchers] = %+v, want union %v", got.Extras["watchers"], want)
	}
}

func TestExtrasDifferentKeysMergeWithoutArbitration(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.Extras = scalarExtras("k1", "x", "k2", "y") })
	ours := base(func(it *datamodel.Item) { it.Extras = scalarExtras("k1", "x2", "k2", "y") })
	theirs := base(func(it *datamodel.Item) { it.Extras = scalarExtras("k1", "x", "k2", "y2") })
	res := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger)
	if v := res.Item.Extras["k1"]; v.Scalar == nil || *v.Scalar != "x2" {
		t.Fatalf("Extras[k1] = %+v, want ours' edit x2", v)
	}
	if v := res.Item.Extras["k2"]; v.Scalar == nil || *v.Scalar != "y2" {
		t.Fatalf("Extras[k2] = %+v, want theirs' edit y2", v)
	}
	if len(res.Arbitrated) != 0 {
		t.Fatalf("Arbitrated = %v, want none for edits to different extras", res.Arbitrated)
	}
}

func TestExtraDeletedOnOneSideStaysDeleted(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.Extras = scalarExtras("team", "infra") })
	ours := base(nil)
	theirs := base(func(it *datamodel.Item) { it.Extras = scalarExtras("team", "infra") })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	if got.Extras != nil {
		t.Fatalf("Extras = %+v, want one-sided deletion to hold", got.Extras)
	}
}
