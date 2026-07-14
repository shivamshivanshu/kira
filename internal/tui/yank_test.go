package tui

import (
	"bytes"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/shivamshivanshu/kira/internal/clipx"
)

func recordingClip() (clipx.Clipboard, *bytes.Buffer) {
	var buf bytes.Buffer
	cb := clipx.Clipboard{
		Getenv:   func(string) string { return "" },
		GOOS:     "linux",
		LookPath: func(string) (string, error) { return "", errors.New("none") },
		Exec:     func(string, []string, []byte) error { return nil },
		Term:     &buf,
	}
	return cb, &buf
}

func yankModel() (model, *bytes.Buffer) {
	m := newTestModel(100, 12, true)
	clip, buf := recordingClip()
	m.clip = clip
	return m, buf
}

func TestYankSelectedID(t *testing.T) {
	t.Parallel()
	m, buf := yankModel()
	updated, _ := m.Update(key("y"))
	if want := clipx.OSC52("E1", false); buf.String() != want {
		t.Fatalf("yank y copied %q, want OSC52 of E1", buf.String())
	}
	if updated.(model).yank != nil {
		t.Error("plain yank must not open the picker")
	}
}

func TestYankPickerCopiesChosenForm(t *testing.T) {
	t.Parallel()
	m, buf := yankModel()

	opened, _ := m.Update(key("Y"))
	m = opened.(model)
	if m.yank == nil {
		t.Fatal("Y did not open the yank picker")
	}
	if m.yank.title != "yank KIRA-100" {
		t.Fatalf("picker titled %q, want the selected epic KIRA-100", m.yank.title)
	}

	chosen, _ := m.Update(key("5"))
	if want := clipx.OSC52("kira-100-order-book-hardening", false); buf.String() != want {
		t.Fatalf("branch form copied %q, want OSC52 of branch name", buf.String())
	}
	if chosen.(model).yank != nil {
		t.Error("selecting a form must close the picker")
	}
}

func TestYankPickerCancels(t *testing.T) {
	t.Parallel()
	m, buf := yankModel()
	opened, _ := m.Update(key("Y"))
	cancelled, _ := opened.(model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cancelled.(model).yank != nil {
		t.Error("esc must close the picker")
	}
	if buf.Len() != 0 {
		t.Errorf("cancel must not copy anything, got %q", buf.String())
	}
}

func TestYankPickerTeatest(t *testing.T) {
	t.Parallel()
	m, buf := yankModel()
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 12))
	tm.Type("Y")
	tm.Type("2")
	tm.Type("q")
	tm.WaitFinished(t)
	if want := clipx.OSC52("KIRA-100", false); buf.String() != want {
		t.Fatalf("picker number form copied %q, want OSC52 of KIRA-100", buf.String())
	}
}
