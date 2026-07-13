package tui

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func initRepo(t *testing.T) (*core.Store, *datamodel.Config, string) {
	t.Helper()
	dir := testutil.InitGitRepo(t)
	if _, err := core.Init(dir, "KIRA", false); err != nil {
		t.Fatalf("core.Init: %v", err)
	}
	s, err := core.Discover(dir)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	cfg, err := s.Config()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	return s, cfg, dir
}

func createTicket(t *testing.T, s *core.Store, cfg *datamodel.Config, title string) string {
	t.Helper()
	res, err := s.Create(cfg, core.CreateOpts{Type: "ticket", Title: title, NoEdit: true})
	if err != nil {
		t.Fatalf("create %q: %v", title, err)
	}
	return res.Number
}
