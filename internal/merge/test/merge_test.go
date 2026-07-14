package merge_test

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/merge"
)

func strptr(s string) *string { return &s }

func base(mut func(*datamodel.Item)) *datamodel.Item {
	it := &datamodel.Item{
		ID:      "01J8X8Q7RZTN5Y3VXW2A9K4E7F",
		Number:  "KIRA-1",
		Type:    datamodel.TypeTicket,
		Title:   "Base title",
		State:   "TODO",
		Created: "2026-01-01T00:00:00Z",
		Updated: "2026-01-01T00:00:00Z",
	}
	if mut != nil {
		mut(it)
	}
	return it
}

func conflictMerger(_, _, _ string) (string, bool) { return "", true }

func gitMerger(b, o, t string) (string, bool) {
	m, c, err := gitx.MergeText(b, o, t)
	if err != nil {
		return "", true
	}
	return m, c
}

func TestScalarOneSidedChangeTaken(t *testing.T) {
	t.Parallel()
	b := base(nil)
	ours := base(nil)
	theirs := base(func(it *datamodel.Item) { it.State = "IN_PROGRESS" })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	if got.State != "IN_PROGRESS" {
		t.Fatalf("state = %q, want IN_PROGRESS (one-sided theirs change)", got.State)
	}
}

func TestScalarBothChangedLaterUpdatedWins(t *testing.T) {
	t.Parallel()
	b := base(nil)
	ours := base(func(it *datamodel.Item) { it.State = "REVIEW"; it.Updated = "2026-02-02T00:00:00Z" })
	theirs := base(func(it *datamodel.Item) { it.State = "DONE"; it.Updated = "2026-02-01T00:00:00Z" })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	if got.State != "REVIEW" {
		t.Fatalf("state = %q, want REVIEW (ours updated later)", got.State)
	}
}

func TestScalarTieBreakGoesToRemote(t *testing.T) {
	t.Parallel()
	b := base(nil)
	ours := base(func(it *datamodel.Item) { it.State = "REVIEW" })
	theirs := base(func(it *datamodel.Item) { it.State = "DONE" })
	if got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item; got.State != "DONE" {
		t.Fatalf("remote=Theirs tie: state = %q, want DONE", got.State)
	}
	if got := merge.Merge(b, ours, theirs, merge.Ours, conflictMerger).Item; got.State != "REVIEW" {
		t.Fatalf("remote=Ours tie: state = %q, want REVIEW", got.State)
	}
}

func TestPointerScalarNilBaseLWW(t *testing.T) {
	t.Parallel()
	b := base(nil)
	ours := base(func(it *datamodel.Item) { it.Owner = strptr("alice"); it.Updated = "2026-02-02T00:00:00Z" })
	theirs := base(func(it *datamodel.Item) { it.Owner = strptr("bob"); it.Updated = "2026-02-01T00:00:00Z" })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	if got.Owner == nil || *got.Owner != "alice" {
		t.Fatalf("owner = %v, want alice", got.Owner)
	}
}

func TestLabelsSetMerge(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.Labels = []string{"a", "b"} })
	ours := base(func(it *datamodel.Item) { it.Labels = []string{"a", "b", "c"} })
	theirs := base(func(it *datamodel.Item) { it.Labels = []string{"b"} })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	want := []string{"b", "c"}
	if !equalStrings(got.Labels, want) {
		t.Fatalf("labels = %v, want %v (b survives both, c added, a removed by theirs)", got.Labels, want)
	}
}

func TestAliasesUnionNeverDrops(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.Aliases = []string{"x"} })
	ours := base(func(it *datamodel.Item) { it.Aliases = nil })
	theirs := base(func(it *datamodel.Item) { it.Aliases = []string{"x", "y"} })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	want := []string{"x", "y"}
	if !equalStrings(got.Aliases, want) {
		t.Fatalf("aliases = %v, want %v (retired numbers never dropped)", got.Aliases, want)
	}
}

func TestLinksUnionByType(t *testing.T) {
	t.Parallel()
	ours := base(func(it *datamodel.Item) { it.Links = map[string][]string{"relates": {"K1"}} })
	theirs := base(func(it *datamodel.Item) { it.Links = map[string][]string{"duplicate_of": {"K2"}} })
	got := merge.Merge(nil, ours, theirs, merge.Theirs, conflictMerger).Item
	if !equalStrings(got.Links["relates"], []string{"K1"}) || !equalStrings(got.Links["duplicate_of"], []string{"K2"}) {
		t.Fatalf("links = %v, want relates=[K1] duplicate_of=[K2]", got.Links)
	}
}

func TestCommentsUnionSortedByTs(t *testing.T) {
	t.Parallel()
	c1 := datamodel.Comment{ID: "c1", Author: "a", Ts: "2026-03-01T00:00:00Z", Body: "first"}
	c2 := datamodel.Comment{ID: "c2", Author: "a", Ts: "2026-03-02T00:00:00Z", Body: "second"}
	ours := base(func(it *datamodel.Item) { it.Body = codec.AppendComment("Prose", c1) })
	theirs := base(func(it *datamodel.Item) { it.Body = codec.AppendComment(codec.AppendComment("Prose", c1), c2) })
	got := merge.Merge(base(func(it *datamodel.Item) { it.Body = "Prose" }), ours, theirs, merge.Theirs, conflictMerger).Item
	parsed := codec.ParseComments(got.Body)
	if len(parsed) != 2 || parsed[0].ID != "c1" || parsed[1].ID != "c2" {
		t.Fatalf("comments = %+v, want [c1 c2] deduped and ts-sorted", parsed)
	}
}

func TestBodyOneSidedChangeTaken(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.Body = "original" })
	ours := base(func(it *datamodel.Item) { it.Body = "original" })
	theirs := base(func(it *datamodel.Item) { it.Body = "rewritten" })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	if got.Body != "rewritten" {
		t.Fatalf("body = %q, want rewritten", got.Body)
	}
}

func TestBodyBothChangedCleanTextMerge(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.Body = "l1\nl2\nl3\n" })
	ours := base(func(it *datamodel.Item) { it.Body = "OURS\nl2\nl3\n" })
	theirs := base(func(it *datamodel.Item) { it.Body = "l1\nl2\nTHEIRS\n" })
	got := merge.Merge(b, ours, theirs, merge.Theirs, gitMerger).Item
	if got.Body != "OURS\nl2\nTHEIRS\n" {
		t.Fatalf("body = %q, want disjoint line merge", got.Body)
	}
}

func TestBodyConflictFallsBackToLWW(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.Body = "original" })
	ours := base(func(it *datamodel.Item) { it.Body = "ours version"; it.Updated = "2026-02-02T00:00:00Z" })
	theirs := base(func(it *datamodel.Item) { it.Body = "theirs version"; it.Updated = "2026-02-01T00:00:00Z" })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	if got.Body != "ours version" {
		t.Fatalf("body = %q, want ours version (LWW on conflict)", got.Body)
	}
}

func TestUpdatedIsMax(t *testing.T) {
	t.Parallel()
	ours := base(func(it *datamodel.Item) { it.Updated = "2026-02-02T00:00:00Z" })
	theirs := base(func(it *datamodel.Item) { it.Updated = "2026-05-05T00:00:00Z" })
	if got := merge.Merge(base(nil), ours, theirs, merge.Theirs, conflictMerger).Item; got.Updated != "2026-05-05T00:00:00Z" {
		t.Fatalf("updated = %q, want max", got.Updated)
	}
}

func TestImmutableFieldsFromOurs(t *testing.T) {
	t.Parallel()
	ours := base(func(it *datamodel.Item) { it.Number = "KIRA-1" })
	theirs := base(func(it *datamodel.Item) { it.Number = "KIRA-9" })
	got := merge.Merge(base(nil), ours, theirs, merge.Theirs, conflictMerger).Item
	if got.Number != "KIRA-1" || got.ID != ours.ID || got.Type != ours.Type || got.Created != ours.Created {
		t.Fatalf("immutable/number not taken from ours: %+v", got)
	}
}

func TestDeterministicAcrossRuns(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.Body = "l1\nl2\nl3\n"; it.Labels = []string{"a"} })
	ours := base(func(it *datamodel.Item) {
		it.Body = "OURS\nl2\nl3\n"
		it.Labels = []string{"a", "c"}
		it.State = "REVIEW"
		it.Updated = "2026-02-02T00:00:00Z"
	})
	theirs := base(func(it *datamodel.Item) {
		it.Body = "l1\nl2\nTHEIRS\n"
		it.Labels = []string{"a", "d"}
		it.State = "DONE"
		it.Updated = "2026-02-01T00:00:00Z"
	})
	first := codec.Serialize(merge.Merge(b, ours, theirs, merge.Theirs, gitMerger).Item)
	second := codec.Serialize(merge.Merge(b, ours, theirs, merge.Theirs, gitMerger).Item)
	if first != second {
		t.Fatalf("non-deterministic:\n%s\n---\n%s", first, second)
	}
}

func TestArbitratedReportsOnlyBothSidesDivergedFromBase(t *testing.T) {
	t.Parallel()
	b := base(nil)
	ours := base(func(it *datamodel.Item) { it.State = "REVIEW"; it.Owner = strptr("alice") })
	theirs := base(func(it *datamodel.Item) { it.State = "DONE"; it.Title = "other" })
	res := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger)
	if !contains(res.Arbitrated, "state") {
		t.Fatalf("arbitrated = %v, want state (both sides diverged from base)", res.Arbitrated)
	}
	if contains(res.Arbitrated, "owner") || contains(res.Arbitrated, "title") {
		t.Fatalf("arbitrated = %v, must exclude one-side-only changes (owner, title)", res.Arbitrated)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
