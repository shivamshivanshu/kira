package core_test

import (
	"errors"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func TestTreeCycleReported(t *testing.T) {
	root := testutil.InitGitRepo(t)
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
	root := testutil.InitGitRepo(t)
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
