package cli

import (
	"os/exec"
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

func TestCommandRunnerFindDropsGlobalFlags(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep not installed")
	}
	dir := initFixture(t)
	s, cfg := reopen(t, dir)
	if _, err := s.Create(cfg, core.CreateOpts{Type: "ticket", Title: "Bridged needle", NoEdit: true}); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Chdir(dir)
	g := &globalFlags{}
	out, err := commandRunner(g)([]string{"find", "needle"})
	if err != nil {
		t.Fatalf("bridged find: %v\n%s", err, out)
	}
	if !strings.Contains(out, "needle") {
		t.Fatalf("runner output = %q, want a match for needle", out)
	}
}

func TestCommandRunnerFindHonoursChdir(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep not installed")
	}
	dir := initFixture(t)
	s, cfg := reopen(t, dir)
	if _, err := s.Create(cfg, core.CreateOpts{Type: "ticket", Title: "Bridged needle", NoEdit: true}); err != nil {
		t.Fatalf("create: %v", err)
	}
	g := &globalFlags{chdir: dir}
	out, err := commandRunner(g)([]string{"find", "needle"})
	if err != nil {
		t.Fatalf("bridged find with chdir set: %v\n%s", err, out)
	}
	if !strings.Contains(out, "needle") {
		t.Fatalf("runner output = %q, want a match for needle", out)
	}
}

func TestCommandRunnerRejectsStdinFromFile(t *testing.T) {
	dir := initFixture(t)
	s, cfg := reopen(t, dir)
	res, err := s.Create(cfg, core.CreateOpts{Type: "ticket", Title: "T", NoEdit: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	g := &globalFlags{chdir: dir}
	for _, argv := range [][]string{
		{"create", "ticket", "--from-file", "-"},
		{"edit", res.Number, "--from-file", "-"},
	} {
		_, err := commandRunner(g)(argv)
		if err == nil || !strings.Contains(err.Error(), "stdin") {
			t.Fatalf("bridged %v = %v, want stdin rejection", argv, err)
		}
	}
}

func TestAutoTUIAllowed(t *testing.T) {
	on := &datamodel.Config{UI: datamodel.UI{AutoTUI: true}}
	off := &datamodel.Config{UI: datamodel.UI{AutoTUI: false}}
	cases := []struct {
		name string
		g    *globalFlags
		cfg  *datamodel.Config
		want bool
	}{
		{"default", &globalFlags{}, on, true},
		{"json", &globalFlags{json: true}, on, false},
		{"non-interactive", &globalFlags{nonInteractive: true}, on, false},
		{"auto_tui off", &globalFlags{}, off, false},
	}
	for _, c := range cases {
		if got := autoTUIAllowed(c.g, c.cfg); got != c.want {
			t.Errorf("autoTUIAllowed(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestCommandRunnerCreateSkipsEditor(t *testing.T) {
	dir := initFixture(t)
	t.Setenv("EDITOR", "false")
	g := &globalFlags{chdir: dir}
	out, err := commandRunner(g)([]string{"create", "ticket", "--title", "Bridged"})
	if err != nil {
		t.Fatalf("bridged create without --no-edit: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Created") {
		t.Fatalf("runner output = %q, want Created", out)
	}
}

func TestCommandRunnerEditWithoutFieldsErrors(t *testing.T) {
	dir := initFixture(t)
	t.Setenv("EDITOR", "false")
	s, cfg := reopen(t, dir)
	res, err := s.Create(cfg, core.CreateOpts{Type: "ticket", Title: "T", NoEdit: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	g := &globalFlags{chdir: dir}
	_, err = commandRunner(g)([]string{"edit", res.Number})
	if err == nil || !strings.Contains(err.Error(), "$EDITOR") {
		t.Fatalf("bridged edit without fields = %v, want editor-unavailable error", err)
	}
}

func TestCommandRunnerCommentWithoutMessageErrors(t *testing.T) {
	dir := initFixture(t)
	t.Setenv("EDITOR", "false")
	s, cfg := reopen(t, dir)
	res, err := s.Create(cfg, core.CreateOpts{Type: "ticket", Title: "T", NoEdit: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	g := &globalFlags{chdir: dir}
	_, err = commandRunner(g)([]string{"comment", res.Number})
	if err == nil || !strings.Contains(err.Error(), "$EDITOR") {
		t.Fatalf("bridged comment without -m = %v, want editor-unavailable error", err)
	}
}

func TestCommandRunnerTUIRefusesNonInteractive(t *testing.T) {
	dir := initFixture(t)
	g := &globalFlags{chdir: dir}
	_, err := commandRunner(g)([]string{"tui"})
	if err == nil || !strings.Contains(err.Error(), "non-interactive") {
		t.Fatalf("bridged tui = %v, want non-interactive refusal", err)
	}
}

func TestCommandRunnerBoardRendersPlainWithAutoTUI(t *testing.T) {
	dir := initFixture(t)
	s, cfg := reopen(t, dir)
	if !cfg.UI.AutoTUI {
		t.Fatal("fixture config should default ui.auto_tui on")
	}
	if _, err := s.Create(cfg, core.CreateOpts{Type: "ticket", Title: "T", NoEdit: true}); err != nil {
		t.Fatalf("create: %v", err)
	}
	g := &globalFlags{chdir: dir}
	out, err := commandRunner(g)([]string{"board"})
	if err != nil {
		t.Fatalf("bridged board: %v\n%s", err, out)
	}
	if !strings.Contains(out, "TODO") {
		t.Fatalf("runner output = %q, want the plain board table", out)
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
