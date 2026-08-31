package core_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestSnapshotMatchesStoreReads(t *testing.T) {
	s, cfg := scopedQueryFixture(t)
	view, err := s.Snapshot(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"", "owner=ORDER ORDER BY rank", "sprint=active"} {
		opts := core.ListOpts{Type: "ticket", Board: "kira", Query: query}
		want, err := s.List(cfg, opts)
		if err != nil {
			t.Fatal(err)
		}
		got, err := view.List(opts)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("snapshot list %q = %+v, %v; want %+v", query, got, err, want)
		}
		boardOpts := core.BoardOpts{Type: opts.Type, Board: opts.Board, Query: query}
		wantBoard, err := s.Board(cfg, boardOpts)
		if err != nil {
			t.Fatal(err)
		}
		gotBoard, err := view.Board(boardOpts)
		if err != nil || !reflect.DeepEqual(gotBoard, wantBoard) {
			t.Fatalf("snapshot board %q = %+v, %v; want %+v", query, gotBoard, err, wantBoard)
		}
	}
	if _, err := view.List(core.ListOpts{At: "HEAD"}); err == nil {
		t.Fatal("snapshot silently ignored a historical list request")
	}
	if _, err := view.Board(core.BoardOpts{At: "HEAD"}); err == nil {
		t.Fatal("snapshot silently ignored a historical board request")
	}
	if _, err := view.List(core.ListOpts{Filter: "ordered", Query: "owner=ORDER ORDER BY rank"}); err == nil {
		t.Fatal("snapshot ignored conflicting query order clauses")
	}
	if _, err := view.Board(core.BoardOpts{Filter: "ordered", Query: "owner=ORDER ORDER BY rank"}); err == nil {
		t.Fatal("snapshot board ignored conflicting query order clauses")
	}
}

func TestSnapshotRetainsItemsAndQueryContext(t *testing.T) {
	s, cfg := scopedQueryFixture(t)
	if _, err := s.SprintActivate(cfg, "s1"); err != nil {
		t.Fatal(err)
	}
	view, err := s.Snapshot(cfg)
	if err != nil {
		t.Fatal(err)
	}
	opts := core.ListOpts{Sprint: "active"}
	before, err := view.List(opts)
	if err != nil {
		t.Fatal(err)
	}
	treeBefore, err := view.Tree("")
	if err != nil {
		t.Fatal(err)
	}
	sprintsBefore := view.SprintList()
	if _, err := s.SprintActivate(cfg, "s2"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: []core.FieldEdit{{Key: "title", Value: "Changed after snapshot"}}}); err != nil {
		t.Fatal(err)
	}
	mustCreate(t, s, cfg, "Created after snapshot")
	after, err := view.List(opts)
	if err != nil || !reflect.DeepEqual(after, before) {
		t.Fatalf("snapshot list changed after a store mutation: %+v, %v", after, err)
	}
	treeAfter, err := view.Tree("")
	if err != nil || !reflect.DeepEqual(treeAfter, treeBefore) {
		t.Fatalf("snapshot tree changed after a store mutation: %+v, %v", treeAfter, err)
	}
	if !reflect.DeepEqual(view.SprintList(), sprintsBefore) {
		t.Fatal("snapshot sprint counts or active sprint changed")
	}
	board, err := view.Board(core.BoardOpts{Sprint: "active"})
	if err != nil {
		t.Fatal(err)
	}
	if got := columnByState(board, "TODO").Items; !reflect.DeepEqual(got, before.Items) {
		t.Fatalf("board and list did not share their item snapshot: %+v", got)
	}
	fresh, err := s.List(cfg, opts)
	if err != nil || reflect.DeepEqual(fresh, before) {
		t.Fatalf("fresh store read did not see the changed active sprint: %+v, %v", fresh, err)
	}
}

func TestSnapshotWarningResultsAreIndependent(t *testing.T) {
	s, cfg := newStore(t)
	view, err := s.Snapshot(cfg)
	if err != nil {
		t.Fatal(err)
	}
	filtered, err := view.List(core.ListOpts{Sprint: "active"})
	if err != nil {
		t.Fatal(err)
	}
	hasNoSprint := func(w datamodel.Warning) bool { return w.Code == datamodel.WarnNoActiveSprint }
	if !slices.ContainsFunc(filtered.StderrNotes, hasNoSprint) {
		t.Fatal("query warning was lost")
	}
	all, err := view.List(core.ListOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(all.StderrNotes, hasNoSprint) {
		t.Fatal("query warnings leaked into another snapshot query")
	}
}

func TestItemDetailUsesOneRevisionAndPreservesSnapshot(t *testing.T) {
	s, cfg := newStore(t)
	it := mustCreate(t, s, cfg, "Before editing")
	view, err := s.Snapshot(cfg)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := view.Detail(it.Number)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Revision != core.ItemRevision(mustReadItem(t, s, it.ID)) || detail.Item.Title != it.Title {
		t.Fatal("item detail and revision do not describe the same file")
	}
	if _, err := s.Edit(cfg, it.ID, core.EditOpts{ExpectedRevision: detail.Revision, Fields: []core.FieldEdit{{Key: "title", Value: "After editing"}}}); err != nil {
		t.Fatal(err)
	}
	if detail.Item.Title != it.Title {
		t.Fatal("editing changed an already-loaded detail")
	}
	if _, err := s.Edit(cfg, it.ID, core.EditOpts{ExpectedRevision: detail.Revision, Fields: []core.FieldEdit{{Key: "title", Value: "Stale edit"}}}); err == nil {
		t.Fatal("stale detail revision was accepted")
	}
	fresh, err := view.Detail(it.ID)
	if err != nil || fresh.Item.Title != "After editing" || fresh.Revision == detail.Revision {
		t.Fatalf("detail must read current file content: %+v, %v", fresh, err)
	}
}
