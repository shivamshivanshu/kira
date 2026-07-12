package core

import (
	"errors"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/item"
)

// treeFixture initializes a repo with an epic (KIRA-1) parenting KIRA-2 and
// KIRA-3, plus an orphan ticket KIRA-4. It returns the store, config, and the
// epic's ULID.
func treeFixture(t *testing.T) (*Store, *config.Config, string) {
	t.Helper()
	root := initGitRepo(t)
	if _, err := Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, _ := Discover(root)
	cfg, _ := s.Config()

	epic, err := s.Create(cfg, CreateOpts{Type: item.TypeEpic, Title: "Epic one", NoEdit: true})
	if err != nil {
		t.Fatalf("create epic: %v", err)
	}
	for _, title := range []string{"child a", "child b"} {
		c, err := s.Create(cfg, CreateOpts{Type: item.TypeTicket, Title: title, NoEdit: true})
		if err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
		if _, err := s.Edit(cfg, c.Number, EditOpts{Fields: []FieldEdit{{Key: "epic", Value: epic.ID}}}); err != nil {
			t.Fatalf("link %s: %v", c.Number, err)
		}
	}
	if _, err := s.Create(cfg, CreateOpts{Type: item.TypeTicket, Title: "orphan", NoEdit: true}); err != nil {
		t.Fatalf("create orphan: %v", err)
	}
	return s, cfg, epic.ID
}

func TestListTreeGrouping(t *testing.T) {
	s, cfg, epicID := treeFixture(t)
	res, err := s.List(cfg, ListOpts{Tree: true})
	if err != nil {
		t.Fatalf("List tree: %v", err)
	}
	if len(res.Tree) != 2 {
		t.Fatalf("groups = %d, want 2 (epic + orphan)", len(res.Tree))
	}
	// First group is the epic (KIRA-1), with its two children.
	g0 := res.Tree[0]
	if g0.Epic == nil || *g0.Epic != epicID {
		t.Fatalf("group 0 epic = %v, want %s", g0.Epic, epicID)
	}
	if g0.EpicNumber == nil || *g0.EpicNumber != "KIRA-1" {
		t.Fatalf("group 0 epic_number = %v, want KIRA-1", g0.EpicNumber)
	}
	if len(g0.Items) != 2 {
		t.Fatalf("epic children = %d, want 2", len(g0.Items))
	}
	// Last group is the orphan bucket (epic null) holding KIRA-4.
	orphan := res.Tree[1]
	if orphan.Epic != nil {
		t.Fatalf("orphan group epic = %v, want null", orphan.Epic)
	}
	if len(orphan.Items) != 1 {
		t.Fatalf("orphan items = %d, want 1", len(orphan.Items))
	}
}

func TestTreeHierarchy(t *testing.T) {
	s, cfg, epicID := treeFixture(t)
	res, err := s.Tree(cfg, "")
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if res.Root != nil {
		t.Fatalf("root = %v, want nil", res.Root)
	}
	// Two roots: the epic and the orphan ticket, epic first (KIRA-1 < KIRA-4).
	if len(res.Nodes) != 2 {
		t.Fatalf("roots = %d, want 2", len(res.Nodes))
	}
	if res.Nodes[0].ID != epicID {
		t.Fatalf("first root = %s, want epic %s", res.Nodes[0].ID, epicID)
	}
	if len(res.Nodes[0].Children) != 2 {
		t.Fatalf("epic children = %d, want 2", len(res.Nodes[0].Children))
	}
	if res.Nodes[0].Children[0].Number != "KIRA-2" {
		t.Fatalf("first child = %s, want KIRA-2", res.Nodes[0].Children[0].Number)
	}
	if len(res.Nodes[1].Children) != 0 {
		t.Fatalf("orphan children = %d, want 0", len(res.Nodes[1].Children))
	}
}

func TestTreeScoped(t *testing.T) {
	s, cfg, epicID := treeFixture(t)
	res, err := s.Tree(cfg, "KIRA-1")
	if err != nil {
		t.Fatalf("Tree KIRA-1: %v", err)
	}
	if res.Root == nil || *res.Root != epicID {
		t.Fatalf("root = %v, want %s", res.Root, epicID)
	}
	if len(res.Nodes) != 1 || res.Nodes[0].ID != epicID {
		t.Fatalf("scoped nodes = %+v, want just the epic", res.Nodes)
	}
	if len(res.Nodes[0].Children) != 2 {
		t.Fatalf("children = %d, want 2", len(res.Nodes[0].Children))
	}
}

// TestTreeCycleReported builds an unrepaired epic cycle (two epics pointing at
// each other) and confirms the scoped traversal reports it as a conflict
// instead of looping forever.
func TestTreeCycleReported(t *testing.T) {
	root := initGitRepo(t)
	if _, err := Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, _ := Discover(root)
	cfg, _ := s.Config()
	a, _ := s.Create(cfg, CreateOpts{Type: item.TypeEpic, Title: "A", NoEdit: true})
	b, _ := s.Create(cfg, CreateOpts{Type: item.TypeEpic, Title: "B", NoEdit: true})
	if _, err := s.Edit(cfg, a.Number, EditOpts{Fields: []FieldEdit{{Key: "epic", Value: b.ID}}}); err != nil {
		t.Fatalf("edit A: %v", err)
	}
	if _, err := s.Edit(cfg, b.Number, EditOpts{Fields: []FieldEdit{{Key: "epic", Value: a.ID}}}); err != nil {
		t.Fatalf("edit B: %v", err)
	}
	_, err := s.Tree(cfg, a.Number)
	if err == nil {
		t.Fatalf("Tree over a cycle returned no error")
	}
	var ce *Error
	if !errors.As(err, &ce) || ce.Code != ExitConflict {
		t.Fatalf("err = %v, want conflict error (exit 2)", err)
	}
}

func TestListQueryAndsWithFlags(t *testing.T) {
	root := initGitRepo(t)
	if _, err := Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, _ := Discover(root)
	cfg, _ := s.Config()
	mk := func(title, owner string) {
		if _, err := s.Create(cfg, CreateOpts{Type: item.TypeTicket, Title: title, Owner: owner, NoEdit: true}); err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
	}
	mk("alpha", "shivam")
	mk("beta", "alice")
	mk("gamma", "shivam")

	// Query alone: owner=shivam matches two.
	q, err := s.List(cfg, ListOpts{Query: "owner=shivam"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if q.Count != 2 {
		t.Fatalf("owner=shivam count = %d, want 2", q.Count)
	}
	// Query ANDed with a flat flag narrows further (title term + owner flag).
	both, err := s.List(cfg, ListOpts{Owner: "shivam", Query: "alpha"})
	if err != nil {
		t.Fatalf("query+flag: %v", err)
	}
	if both.Count != 1 || both.Items[0].Title != "alpha" {
		t.Fatalf("owner=shivam AND title~alpha = %+v, want just alpha", both.Items)
	}
	// A malformed query is a user error.
	if _, err := s.List(cfg, ListOpts{Query: "state="}); err == nil {
		t.Fatalf("malformed query returned no error")
	}
}
