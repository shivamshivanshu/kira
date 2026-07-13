package cli

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func TestCommandRunnerMovesTicketThroughCoreService(t *testing.T) {
	dir := initFixture(t)
	s, cfg := reopen(t, dir)
	res, err := s.Create(cfg, core.CreateOpts{Type: "ticket", Title: "T", NoEdit: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	g := &globalFlags{chdir: dir}
	out, err := commandRunner(g)([]string{"move", res.Number, "IN_PROGRESS"})
	if err != nil {
		t.Fatalf("runner move: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Moved "+res.Number) {
		t.Fatalf("runner output = %q, want Moved %s", out, res.Number)
	}

	s2, cfg2 := reopen(t, dir)
	if got := stateOf(t, s2, cfg2, res.Number); got != "IN_PROGRESS" {
		t.Fatalf("%s state = %q, want IN_PROGRESS (same core.Move path as CLI)", res.Number, got)
	}
}

func TestCommandRunnerReportsError(t *testing.T) {
	dir := initFixture(t)
	g := &globalFlags{chdir: dir}
	_, err := commandRunner(g)([]string{"move", "KIRA-999", "IN_PROGRESS"})
	if err == nil {
		t.Fatal("moving an unknown ticket should error")
	}
}

func initFixture(t *testing.T) string {
	t.Helper()
	dir := testutil.InitGitRepo(t)
	if _, err := core.Init(dir, "KIRA", false); err != nil {
		t.Fatalf("core.Init: %v", err)
	}
	return dir
}

func reopen(t *testing.T, dir string) (*core.Store, *datamodel.Config) {
	t.Helper()
	s, err := core.Discover(dir)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	cfg, err := s.Config()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	return s, cfg
}

func stateOf(t *testing.T, s *core.Store, cfg *datamodel.Config, number string) string {
	t.Helper()
	res, err := s.List(cfg, core.ListOpts{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, it := range res.Items {
		if it.Number == number {
			return it.State
		}
	}
	t.Fatalf("ticket %s not found", number)
	return ""
}
