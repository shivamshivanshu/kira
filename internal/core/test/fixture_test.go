package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/storage"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func writeTempItem(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "item.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func newStore(t *testing.T) (*core.Store, *datamodel.Config) {
	t.Helper()
	root := testutil.InitGitRepo(t)
	if _, err := core.Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, err := core.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	cfg, err := s.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	return s, cfg
}

func mustCreate(t *testing.T, s *core.Store, cfg *datamodel.Config, title string) *datamodel.CreateResult {
	t.Helper()
	res, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: title, NoEdit: true})
	if err != nil {
		t.Fatalf("Create %q: %v", title, err)
	}
	return res
}

func stateOf(t *testing.T, s *core.Store, cfg *datamodel.Config, ref string) string {
	t.Helper()
	show, err := s.Show(cfg, ref, "")
	if err != nil {
		t.Fatalf("Show %s: %v", ref, err)
	}
	return show.State
}

func positionTo(t *testing.T, s *core.Store, cfg *datamodel.Config, ref, state string) {
	t.Helper()
	if _, err := s.Move(cfg, ref, state, core.MoveOpts{Force: true}); err != nil {
		t.Fatalf("position %s to %s: %v", ref, state, err)
	}
}

func withTicketTransitions(cfg *datamodel.Config, from string, ts []datamodel.Transition) {
	cfg.Workflows[datamodel.TypeTicket].Transitions[from] = ts
}

func mustReadItem(t *testing.T, s *core.Store, ulid string) string {
	t.Helper()
	data, err := os.ReadFile(storage.New(s.Root()).ItemPath(ulid))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
