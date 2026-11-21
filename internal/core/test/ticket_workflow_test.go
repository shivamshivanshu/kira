package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

const customTicketWorkflows = `
ticket_states:
  bug:
    - {name: Ready, category: todo}
    - {name: Fixing, category: doing}
    - {name: Verified, category: done}
    - {name: Archived, category: done}
  task:
    - {name: Ready, category: doing}
    - {name: Shipped, category: done}
`

func enableTicketWorkflows(t *testing.T, s *core.Store) *datamodel.Config {
	t.Helper()
	path := filepath.Join(s.Root(), ".kira", "config.yaml")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(original, customTicketWorkflows...), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := s.Config()
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestCustomTicketWorkflowTransitionsAndMixedBoard(t *testing.T) {
	s, _ := newStore(t)
	cfg := enableTicketWorkflows(t, s)
	bug, err := s.Create(cfg, core.CreateOpts{
		Type: datamodel.TypeTicket, Title: "bug", NoEdit: true,
		Fields: []core.FieldEdit{{Key: "subtype", Value: "bug"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	task, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Subtype: "task", Title: "task", NoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	if bug.State != "Ready" || bug.Category != "todo" || task.State != "Ready" || task.Category != "doing" {
		t.Fatalf("initial bug = %s/%s, task = %s/%s", bug.State, bug.Category, task.State, task.Category)
	}
	plain := mustCreate(t, s, cfg, "plain")
	if plain.State != "TODO" {
		t.Fatalf("ordinary ticket starts in %s", plain.State)
	}

	board, err := s.Board(cfg, core.BoardOpts{})
	if err != nil {
		t.Fatal(err)
	}
	ready := map[string]datamodel.BoardColumn{}
	for _, column := range board.Columns {
		if column.State == "Ready" {
			ready[column.Category] = column
		}
	}
	for _, result := range []*datamodel.CreateResult{bug, task} {
		column := ready[result.Category]
		if column.Count != 1 || len(column.Items) != 1 || column.Items[0].ID != result.ID {
			t.Fatalf("Ready/%s column = %+v", result.Category, column)
		}
	}

	if _, err := s.Link(cfg, bug.ID, core.LinkOpts{Target: core.LinkBlockedBy, Ref: task.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Move(cfg, bug.ID, "Verified", core.MoveOpts{}); err == nil || !strings.Contains(err.Error(), task.Number) {
		t.Fatalf("expected blocker %s, got %v", task.Number, err)
	}
	if _, err := s.Move(cfg, task.ID, "Shipped", core.MoveOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Move(cfg, bug.ID, "Verified", core.MoveOpts{}); err != nil {
		t.Fatal(err)
	}
	closed, err := s.Show(cfg, bug.ID, "")
	if err != nil || closed.Category != "done" || closed.Resolution == nil || *closed.Resolution != "done" {
		t.Fatalf("closed bug = %+v, %v", closed, err)
	}
	if _, err := s.Move(cfg, bug.ID, "Verified", core.MoveOpts{Resolution: "duplicate"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Edit(cfg, bug.ID, core.EditOpts{Fields: []core.FieldEdit{
		{Key: "state", Value: "Archived"}, {Key: "resolution", Value: "duplicate"},
	}}); err != nil {
		t.Fatal(err)
	}
	archived, err := s.Show(cfg, bug.ID, "")
	if err != nil || archived.Resolution == nil || *archived.Resolution != "duplicate" {
		t.Fatalf("archived bug = %+v, %v", archived, err)
	}
	if _, err := s.Move(cfg, bug.ID, "DONE", core.MoveOpts{}); err == nil {
		t.Fatal("accepted an undeclared default state")
	}
	if _, err := s.Edit(cfg, bug.ID, core.EditOpts{Fields: []core.FieldEdit{{Key: "subtype", Value: "task"}}}); err == nil {
		t.Fatal("accepted subtype change leaving an invalid state")
	}
	if _, err := s.Edit(cfg, bug.ID, core.EditOpts{Fields: []core.FieldEdit{
		{Key: "subtype", Value: "task"}, {Key: "state", Value: "Ready"},
	}}); err != nil {
		t.Fatal(err)
	}
	reopened, err := s.Show(cfg, bug.ID, "")
	if err != nil || reopened.Category != "doing" || reopened.Resolution != nil {
		t.Fatalf("reopened bug = %+v, %v", reopened, err)
	}
}

func TestLegacyTicketStateRemainsVisibleAndMovable(t *testing.T) {
	s, cfg := newStore(t)
	bug, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Subtype: "bug", Title: "old bug", NoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	cfg = enableTicketWorkflows(t, s)
	board, err := s.Board(cfg, core.BoardOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if column := columnByState(board, "TODO"); len(column.Items) != 1 || column.Items[0].ID != bug.ID {
		t.Fatalf("legacy TODO is absent from board: %+v", board)
	}
	if _, err := s.Edit(cfg, bug.ID, core.EditOpts{Fields: []core.FieldEdit{{Key: "title", Value: "still visible"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Move(cfg, bug.ID, "Fixing", core.MoveOpts{}); err != nil {
		t.Fatal(err)
	}
	old, err := s.Show(cfg, bug.ID, "HEAD~2")
	if err != nil || old.State != "TODO" || old.Category != "todo" {
		t.Fatalf("historical bug = %+v, %v", old, err)
	}
}

func TestBoardHidesWipWhenLegacyCustomTicketSharesLane(t *testing.T) {
	s, cfg := newStore(t)
	bug, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Subtype: "bug", Title: "old bug", NoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Move(cfg, bug.ID, "REVIEW", core.MoveOpts{}); err != nil {
		t.Fatal(err)
	}
	mustCreate(t, s, cfg, "plain ticket")
	board, err := s.Board(enableTicketWorkflows(t, s), core.BoardOpts{})
	if err != nil {
		t.Fatal(err)
	}
	review := columnByState(board, "REVIEW")
	if review.Wip != 0 || len(review.Items) != 1 || review.Items[0].ID != bug.ID {
		t.Fatalf("legacy REVIEW column = %+v", review)
	}
}
