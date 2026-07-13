package core_test

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func columnByState(res *datamodel.BoardResult, state string) datamodel.BoardColumn {
	for _, c := range res.Columns {
		if c.State == state {
			return c
		}
	}
	return datamodel.BoardColumn{}
}

func TestBoardColumnsFollowWorkflowOrder(t *testing.T) {
	s, cfg := newStore(t)
	res, err := s.Board(cfg, core.BoardOpts{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"TODO", "IN_PROGRESS", "REVIEW", "DONE", "WONT_DO"}
	got := make([]string, len(res.Columns))
	for i, c := range res.Columns {
		got[i] = c.State
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("columns = %v, want %v", got, want)
	}
	if ip := columnByState(res, "IN_PROGRESS"); ip.Wip != 3 {
		t.Errorf("IN_PROGRESS wip = %d, want 3", ip.Wip)
	}
}

func TestBoardBucketsAndGlobalCounts(t *testing.T) {
	s, cfg := newStore(t)
	a := mustCreate(t, s, cfg, "a")
	mustCreate(t, s, cfg, "b")
	if _, err := s.Move(cfg, a.ID, "IN_PROGRESS", core.MoveOpts{}); err != nil {
		t.Fatal(err)
	}

	res, err := s.Board(cfg, core.BoardOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if todo := columnByState(res, "TODO"); len(todo.Items) != 1 || todo.Count != 1 {
		t.Errorf("TODO items=%d count=%d, want 1/1", len(todo.Items), todo.Count)
	}
	if ip := columnByState(res, "IN_PROGRESS"); len(ip.Items) != 1 || ip.Items[0].ID != a.ID {
		t.Errorf("IN_PROGRESS should hold the moved ticket, got %+v", ip.Items)
	}
}

func TestBoardCountsAreGlobalWhileItemsAreEpicScoped(t *testing.T) {
	s, cfg := newStore(t)
	epic, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeEpic, Title: "E", NoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	scoped, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "under", Parent: epic.ID, NoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	loose := mustCreate(t, s, cfg, "loose")
	for _, id := range []string{scoped.ID, loose.ID} {
		if _, err := s.Move(cfg, id, "IN_PROGRESS", core.MoveOpts{}); err != nil {
			t.Fatal(err)
		}
	}

	res, err := s.Board(cfg, core.BoardOpts{Epic: epic.ID})
	if err != nil {
		t.Fatal(err)
	}
	ip := columnByState(res, "IN_PROGRESS")
	if len(ip.Items) != 1 || ip.Items[0].ID != scoped.ID {
		t.Errorf("epic-scoped IN_PROGRESS should show only the epic's ticket, got %+v", ip.Items)
	}
	if ip.Count != 2 {
		t.Errorf("global IN_PROGRESS count = %d, want 2 (must not understate real column pressure)", ip.Count)
	}
}

func TestBoardAtIsSeamedOff(t *testing.T) {
	s, cfg := newStore(t)
	_, err := s.Board(cfg, core.BoardOpts{At: "HEAD"})
	if err == nil || !strings.Contains(err.Error(), "requires the M3 tree-ish loader") {
		t.Fatalf("board --at should return a clear seam error, got %v", err)
	}
}

func TestAdjacentAllowedMirrorsTransitionGraph(t *testing.T) {
	_, cfg := newStore(t)
	cases := []struct {
		from, to string
		want     bool
	}{
		{"TODO", "IN_PROGRESS", true},
		{"IN_PROGRESS", "REVIEW", true},
		{"DONE", "WONT_DO", false},
		{"TODO", "DONE", false},
	}
	for _, c := range cases {
		if got := core.AdjacentAllowed(cfg, datamodel.TypeTicket, c.from, c.to); got != c.want {
			t.Errorf("AdjacentAllowed(%s->%s) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}
