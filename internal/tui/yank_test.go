package tui

import (
	"bytes"
	"errors"
	"strings"
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
	if updated.(model).picker != nil {
		t.Error("plain yank must not open the picker")
	}
}

func TestYankPickerCopiesChosenForm(t *testing.T) {
	t.Parallel()
	m, buf := yankModel()

	opened, _ := m.Update(key("Y"))
	m = opened.(model)
	if m.picker == nil {
		t.Fatal("Y did not open the yank picker")
	}
	if m.picker.title != "yank KIRA-100" {
		t.Fatalf("picker titled %q, want the selected epic KIRA-100", m.picker.title)
	}

	chosen, _ := m.Update(key("5"))
	if want := clipx.OSC52("kira-100-order-book-hardening", false); buf.String() != want {
		t.Fatalf("branch form copied %q, want OSC52 of branch name", buf.String())
	}
	if chosen.(model).picker != nil {
		t.Error("selecting a form must close the picker")
	}
}

func boardYankModel() (model, *bytes.Buffer) {
	m, buf := yankModel()
	bs := m.screens[viewBoard].(*boardScreen)
	bs.loaded = true
	bs.board.load(buildBoardResult())
	bs.board.focusByID("t2")
	m.view = viewBoard
	return m, buf
}

func TestYankOnBoardCopiesSelectedCard(t *testing.T) {
	t.Parallel()
	m, buf := boardYankModel()
	m.Update(key("y"))
	if want := clipx.OSC52("t2", false); buf.String() != want {
		t.Fatalf("yank on board copied %q, want OSC52 of the selected card id t2", buf.String())
	}
}

func TestYankPickerOnBoardTargetsSelectedCard(t *testing.T) {
	t.Parallel()
	m, _ := boardYankModel()
	opened, _ := m.Update(key("Y"))
	yank := opened.(model).picker
	if yank == nil || yank.title != "yank KIRA-142" {
		t.Fatalf("Y on board should pick from the selected card KIRA-142, got %+v", yank)
	}
}

func TestStatsHintOmitsYank(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	m.view = viewStats
	if strings.Contains(hintLine(m.activeKeys()), "yank") {
		t.Fatal("stats has nothing to yank; y/Y must not be advertised")
	}
	m.view = viewBoard
	if !strings.Contains(hintLine(m.activeKeys()), "yank") {
		t.Fatal("board hints must keep y/Y")
	}
}

func TestYankPickerCancels(t *testing.T) {
	t.Parallel()
	m, buf := yankModel()
	opened, _ := m.Update(key("Y"))
	cancelled, _ := opened.(model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cancelled.(model).picker != nil {
		t.Error("esc must close the picker")
	}
	if buf.Len() != 0 {
		t.Errorf("cancel must not copy anything, got %q", buf.String())
	}
}

func failingClip() clipx.Clipboard {
	return clipx.Clipboard{
		Getenv:   func(k string) string { return map[string]string{"DISPLAY": ":0"}[k] },
		GOOS:     "linux",
		LookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil },
		Exec:     func(string, []string, []byte) error { return errors.New("exit status 1") },
	}
}

func TestYankCopyFailureFlashesError(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	m.clip = failingClip()
	updated, _ := m.Update(key("y"))
	m2 := updated.(model)
	if !m2.bar.msgErr {
		t.Fatal("a failed copy must flash as an error")
	}
	if !strings.Contains(m2.bar.msg, "copy: ") || !strings.Contains(m2.bar.msg, "xclip") {
		t.Fatalf("error flash must name the failing tool, got %q", m2.bar.msg)
	}
	if !strings.Contains(m2.View(), m2.bar.msg) {
		t.Fatal("the copy error must be visible in the footer")
	}
}

func TestYankCopySuccessStaysSilent(t *testing.T) {
	t.Parallel()
	m, _ := yankModel()
	updated, _ := m.Update(key("y"))
	if got := updated.(model).bar.msg; got != "" {
		t.Fatalf("a successful copy must not flash, got %q", got)
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
