package core_test

import (
	"errors"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func treeFixture(t *testing.T) (*core.Store, *datamodel.Config, string) {
	t.Helper()
	root := initGitRepo(t)
	if _, err := core.Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, _ := core.Discover(root)
	cfg, _ := s.Config()

	epic, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeEpic, Title: "Epic one", NoEdit: true})
	if err != nil {
		t.Fatalf("create epic: %v", err)
	}
	for _, title := range []string{"child a", "child b"} {
		c, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: title, NoEdit: true})
		if err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
		if _, err := s.Edit(cfg, c.Number, core.EditOpts{Fields: []core.FieldEdit{{Key: "epic", Value: epic.ID}}}); err != nil {
			t.Fatalf("link %s: %v", c.Number, err)
		}
	}
	if _, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "orphan", NoEdit: true}); err != nil {
		t.Fatalf("create orphan: %v", err)
	}
	return s, cfg, epic.ID
}

func TestListTreeGrouping(t *testing.T) {
	s, cfg, epicID := treeFixture(t)
	res, err := s.List(cfg, core.ListOpts{Tree: true})
	if err != nil {
		t.Fatalf("List tree: %v", err)
	}
	if len(res.Tree) != 2 {
		t.Fatalf("groups = %d, want 2 (epic + orphan)", len(res.Tree))
	}
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
	res, err := s.Tree(cfg, "", "")
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if res.Root != nil {
		t.Fatalf("root = %v, want nil", res.Root)
	}
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
	res, err := s.Tree(cfg, "KIRA-1", "")
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

func TestTreeCycleReported(t *testing.T) {
	root := initGitRepo(t)
	if _, err := core.Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, _ := core.Discover(root)
	cfg, _ := s.Config()
	a, _ := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeEpic, Title: "A", NoEdit: true})
	b, _ := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeEpic, Title: "B", NoEdit: true})
	if _, err := s.Edit(cfg, a.Number, core.EditOpts{Fields: []core.FieldEdit{{Key: "epic", Value: b.ID}}}); err != nil {
		t.Fatalf("edit A: %v", err)
	}
	if _, err := s.Edit(cfg, b.Number, core.EditOpts{Fields: []core.FieldEdit{{Key: "epic", Value: a.ID}}}); err != nil {
		t.Fatalf("edit B: %v", err)
	}
	_, err := s.Tree(cfg, a.Number, "")
	if err == nil {
		t.Fatalf("Tree over a cycle returned no error")
	}
	var ce *errx.Error
	if !errors.As(err, &ce) || ce.Code != errx.ExitConflict {
		t.Fatalf("err = %v, want conflict error (exit 2)", err)
	}
}

func TestListQueryAndsWithFlags(t *testing.T) {
	root := initGitRepo(t)
	if _, err := core.Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, _ := core.Discover(root)
	cfg, _ := s.Config()
	mk := func(title, owner string) {
		if _, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: title, Owner: owner, NoEdit: true}); err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
	}
	mk("alpha", "shivam")
	mk("beta", "alice")
	mk("gamma", "shivam")

	q, err := s.List(cfg, core.ListOpts{Query: "owner=shivam"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if q.Count != 2 {
		t.Fatalf("owner=shivam count = %d, want 2", q.Count)
	}
	both, err := s.List(cfg, core.ListOpts{Owner: "shivam", Query: "alpha"})
	if err != nil {
		t.Fatalf("query+flag: %v", err)
	}
	if both.Count != 1 || both.Items[0].Title != "alpha" {
		t.Fatalf("owner=shivam AND title~alpha = %+v, want just alpha", both.Items)
	}
	if _, err := s.List(cfg, core.ListOpts{Query: "state="}); err == nil {
		t.Fatalf("malformed query returned no error")
	}
}
