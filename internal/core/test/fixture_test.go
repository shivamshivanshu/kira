package core_test

import (
	"os"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func initGitRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("EDITOR", "true")
	root := t.TempDir()
	repo := gitx.Repo{Dir: root}
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "tester"},
	} {
		if _, err := repo.Output(args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	return root
}

func newStore(t *testing.T) (*core.Store, *datamodel.Config) {
	t.Helper()
	root := initGitRepo(t)
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
	if _, err := s.Edit(cfg, ref, core.EditOpts{Fields: []core.FieldEdit{{Key: "state", Value: state}}}); err != nil {
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
