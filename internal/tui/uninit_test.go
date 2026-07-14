package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func TestUninitTypedConfirmCreatesKira(t *testing.T) {
	t.Parallel()
	dir := testutil.InitGitRepo(t)
	m := newUninitModel(dir, asciiTheme())

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 20))
	tm.Type("KIRA")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(10*time.Second))

	final, ok := tm.FinalModel(t).(uninitModel)
	if !ok {
		t.Fatalf("final model type = %T", tm.FinalModel(t))
	}
	if !final.initialized {
		t.Fatalf("model should report initialized; msg = %q", final.msg)
	}
	if fi, err := os.Stat(filepath.Join(dir, ".kira")); err != nil || !fi.IsDir() {
		t.Fatalf(".kira not created: %v", err)
	}
	s, err := core.Discover(dir)
	if err != nil {
		t.Fatalf("discover after init: %v", err)
	}
	cfg, err := s.Config()
	if err != nil {
		t.Fatalf("config after init: %v", err)
	}
	if cfg.Project.Key != "KIRA" {
		t.Fatalf("project key = %q, want KIRA", cfg.Project.Key)
	}
}

func TestUninitEmptyKeyDoesNotInit(t *testing.T) {
	t.Parallel()
	dir := testutil.InitGitRepo(t)
	m := newUninitModel(dir, asciiTheme())
	m.width, m.height = 80, 20

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um := updated.(uninitModel)
	if um.initialized {
		t.Fatal("empty key must not initialize")
	}
	if _, err := os.Stat(filepath.Join(dir, ".kira")); !os.IsNotExist(err) {
		t.Fatalf(".kira should not exist after empty confirm: %v", err)
	}
}
