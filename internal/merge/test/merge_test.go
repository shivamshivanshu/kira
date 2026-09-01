package merge_test

import (
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/merge"
)

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
	ours := base(func(it *datamodel.Item) { it.Owner = new("alice"); it.Updated = "2026-02-02T00:00:00Z" })
	theirs := base(func(it *datamodel.Item) { it.Owner = new("bob"); it.Updated = "2026-02-01T00:00:00Z" })
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

func TestExtraLinksUnionByType(t *testing.T) {
	t.Parallel()
	ours := base(func(it *datamodel.Item) { it.ExtraLinks = map[string][]string{"mystery": {"K1"}} })
	theirs := base(func(it *datamodel.Item) { it.ExtraLinks = map[string][]string{"future_rel": {"K2"}} })
	got := merge.Merge(nil, ours, theirs, merge.Theirs, conflictMerger).Item
	if !equalStrings(got.ExtraLinks["mystery"], []string{"K1"}) || !equalStrings(got.ExtraLinks["future_rel"], []string{"K2"}) {
		t.Fatalf("ExtraLinks = %v, want mystery=[K1] future_rel=[K2]", got.ExtraLinks)
	}
}

func TestExtraScalarAddedOnOneSideSurvives(t *testing.T) {
	t.Parallel()
	b := base(nil)
	ours := base(nil)
	theirs := base(func(it *datamodel.Item) { it.ExtraScalars = map[string]string{"severity": "HIGH"} })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	if got.ExtraScalars["severity"] != "HIGH" {
		t.Fatalf("ExtraScalars = %v, want severity=HIGH (added on theirs, must survive)", got.ExtraScalars)
	}
}

func TestExtraScalarBothChangedLaterUpdatedWins(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.ExtraScalars = map[string]string{"severity": "LOW"} })
	ours := base(func(it *datamodel.Item) {
		it.ExtraScalars = map[string]string{"severity": "HIGH"}
		it.Updated = "2026-02-02T00:00:00Z"
	})
	theirs := base(func(it *datamodel.Item) {
		it.ExtraScalars = map[string]string{"severity": "MEDIUM"}
		it.Updated = "2026-02-01T00:00:00Z"
	})
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	if got.ExtraScalars["severity"] != "HIGH" {
		t.Fatalf("ExtraScalars[severity] = %q, want HIGH (ours updated later)", got.ExtraScalars["severity"])
	}
}

func TestExtraListUnion(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.ExtraLists = map[string][]string{"watchers": {"a", "b"}} })
	ours := base(func(it *datamodel.Item) { it.ExtraLists = map[string][]string{"watchers": {"a", "b", "c"}} })
	theirs := base(func(it *datamodel.Item) { it.ExtraLists = map[string][]string{"watchers": {"b"}} })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	want := []string{"b", "c"}
	if !equalStrings(got.ExtraLists["watchers"], want) {
		t.Fatalf("ExtraLists[watchers] = %v, want %v (b survives both, c added, a removed by theirs)", got.ExtraLists["watchers"], want)
	}
}

func TestExtraShapeCollisionWinnerWinsWholesale(t *testing.T) {
	t.Parallel()
	b := base(nil)
	ours := base(func(it *datamodel.Item) {
		it.ExtraScalars = map[string]string{"custom": "scalar-value"}
		it.Updated = "2026-02-02T00:00:00Z"
	})
	theirs := base(func(it *datamodel.Item) {
		it.ExtraLists = map[string][]string{"custom": {"a", "b"}}
		it.Updated = "2026-02-01T00:00:00Z"
	})
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item

	if _, isList := got.ExtraLists["custom"]; isList {
		t.Fatalf("custom must not survive in both maps: scalars=%v lists=%v", got.ExtraScalars, got.ExtraLists)
	}
	if got.ExtraScalars["custom"] != "scalar-value" {
		t.Fatalf("ExtraScalars[custom] = %q, want scalar-value (ours updated later, wins wholesale)", got.ExtraScalars["custom"])
	}

	out := codec.Serialize(got)
	if _, err := codec.Parse(out); err != nil {
		t.Fatalf("shape-collision merge output must re-parse: %v\n%s", err, out)
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

func TestFreshEmptyBodyCommentUnionKeepsBoth(t *testing.T) {
	t.Parallel()
	fresh := base(func(it *datamodel.Item) { it.Body = "" })
	doc := codec.Serialize(fresh)
	cA := datamodel.Comment{ID: "ca", Author: "alice", Ts: "2026-03-01T00:00:00Z", Body: "from branch A"}
	cB := datamodel.Comment{ID: "cb", Author: "bob", Ts: "2026-03-02T00:00:00Z", Body: "from branch B"}

	side := func(c datamodel.Comment) *datamodel.Item {
		t.Helper()
		withComment, err := codec.AppendCommentToDocument(doc, c)
		if err != nil {
			t.Fatalf("append: %v", err)
		}
		it, err := codec.Parse(withComment)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		return it
	}
	got := merge.Merge(fresh, side(cA), side(cB), merge.Theirs, gitMerger).Item
	parsed := codec.ParseComments(got.Body)
	if len(parsed) != 2 || parsed[0].ID != "ca" || parsed[1].ID != "cb" {
		t.Fatalf("comments = %+v, want both branch comments to survive the union merge", parsed)
	}
}

func TestBodyNonCanonicalCommentsFallToTextMerge(t *testing.T) {
	t.Parallel()
	c1 := datamodel.Comment{ID: "c1", Author: "a", Ts: "2026-03-01T00:00:00Z", Body: "first"}
	canonical := codec.AppendComment("Prose", c1)
	b := base(func(it *datamodel.Item) { it.Body = canonical })
	ours := base(func(it *datamodel.Item) { it.Body = canonical })
	theirs := base(func(it *datamodel.Item) { it.Body = canonical + "epilogue after the comment\n" })
	got := merge.Merge(b, ours, theirs, merge.Theirs, gitMerger).Item
	if got.Body != theirs.Body {
		t.Fatalf("one-sided epilogue must survive the merge:\nwant %q\ngot  %q", theirs.Body, got.Body)
	}
	if parsed := codec.ParseComments(got.Body); len(parsed) != 1 || parsed[0].ID != "c1" {
		t.Fatalf("comment must not duplicate or vanish, got %+v", parsed)
	}
}

func TestBodyUnterminatedCommentNotDropped(t *testing.T) {
	t.Parallel()
	body := "Prose\n<!-- kira:comment id=c9 author=a ts=2026-03-01T00:00:00Z -->\ndangling\n"
	b := base(func(it *datamodel.Item) { it.Body = "Prose\n" })
	ours := base(func(it *datamodel.Item) { it.Body = "Prose\n" })
	theirs := base(func(it *datamodel.Item) { it.Body = body })
	got := merge.Merge(b, ours, theirs, merge.Theirs, gitMerger).Item
	if got.Body != body {
		t.Fatalf("unterminated comment must merge as plain text, not vanish:\nwant %q\ngot  %q", body, got.Body)
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
	ours := base(nil)
	theirs := base(func(it *datamodel.Item) {
		it.ID = "01J8X8Q7RZTN5Y3VXW2A9K4E99"
		it.Created = "2027-01-01T00:00:00Z"
	})
	got := merge.Merge(base(nil), ours, theirs, merge.Theirs, conflictMerger).Item
	if got.ID != ours.ID || got.Type != ours.Type || got.Created != ours.Created {
		t.Fatalf("immutable fields not taken from ours: %+v", got)
	}
}

func TestNumberRemoteBoardMoveSurvivesMerge(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.Number = "DX-7" })
	ours := base(func(it *datamodel.Item) { it.Number = "DX-7"; it.Title = "Edited locally" })
	theirs := base(func(it *datamodel.Item) { it.Number = "CORE-3"; it.Aliases = []string{"DX-7"} })
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	if got.Number != "CORE-3" || got.Title != "Edited locally" {
		t.Fatalf("number = %q title = %q, want one-sided renumber CORE-3 with local title", got.Number, got.Title)
	}
	if !equalStrings(got.Aliases, []string{"DX-7"}) {
		t.Fatalf("aliases = %v, want [DX-7] and never the item's own number", got.Aliases)
	}
	rev := merge.Merge(b, theirs, ours, merge.Ours, conflictMerger).Item
	if rev.Number != "CORE-3" || rev.Title != "Edited locally" || !equalStrings(rev.Aliases, []string{"DX-7"}) {
		t.Fatalf("mirror direction: number = %q title = %q aliases = %v, want CORE-3 / local title / [DX-7]", rev.Number, rev.Title, rev.Aliases)
	}
}

func TestNumberBothRenumberedLaterUpdatedWins(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.Number = "DX-7" })
	ours := base(func(it *datamodel.Item) {
		it.Number = "CORE-3"
		it.Aliases = []string{"DX-7"}
		it.Updated = "2026-02-01T00:00:00Z"
	})
	theirs := base(func(it *datamodel.Item) {
		it.Number = "CLI-2"
		it.Aliases = []string{"DX-7"}
		it.Updated = "2026-02-02T00:00:00Z"
	})
	got := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger).Item
	if got.Number != "CLI-2" {
		t.Fatalf("number = %q, want CLI-2 (theirs updated later)", got.Number)
	}
	if !equalStrings(got.Aliases, []string{"DX-7"}) {
		t.Fatalf("aliases = %v, want [DX-7]", got.Aliases)
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
	ours := base(func(it *datamodel.Item) { it.State = "REVIEW"; it.Owner = new("alice") })
	theirs := base(func(it *datamodel.Item) { it.State = "DONE"; it.Title = "other" })
	res := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger)
	if !slices.Contains(res.Arbitrated, "state") {
		t.Fatalf("arbitrated = %v, want state (both sides diverged from base)", res.Arbitrated)
	}
	if slices.Contains(res.Arbitrated, "owner") || slices.Contains(res.Arbitrated, "title") {
		t.Fatalf("arbitrated = %v, must exclude one-side-only changes (owner, title)", res.Arbitrated)
	}
}

func TestArbitratedReportsExtrasOnlyConflict(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) { it.ExtraScalars = map[string]string{"severity": "LOW"} })
	ours := base(func(it *datamodel.Item) { it.ExtraScalars = map[string]string{"severity": "HIGH"} })
	theirs := base(func(it *datamodel.Item) { it.ExtraScalars = map[string]string{"severity": "MEDIUM"} })
	res := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger)
	if !slices.Contains(res.Arbitrated, "severity") {
		t.Fatalf("arbitrated = %v, want severity (both sides changed the same extra from base)", res.Arbitrated)
	}
}

func TestArbitratedReportsExtraLinkConflict(t *testing.T) {
	t.Parallel()
	b := base(func(it *datamodel.Item) {
		it.ExtraLinks = map[string][]string{"mirrors": {"01HAAAA000000000000000000A"}}
	})
	ours := base(func(it *datamodel.Item) {
		it.ExtraLinks = map[string][]string{"mirrors": {"01HAAAA000000000000000000B"}}
	})
	theirs := base(func(it *datamodel.Item) {
		it.ExtraLinks = map[string][]string{"mirrors": {"01HAAAA000000000000000000C"}}
	})
	res := merge.Merge(b, ours, theirs, merge.Theirs, conflictMerger)
	if !slices.Contains(res.Arbitrated, datamodel.KeyLinks) {
		t.Fatalf("arbitrated = %v, want %q (both sides diverged the same unknown link type)", res.Arbitrated, datamodel.KeyLinks)
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
