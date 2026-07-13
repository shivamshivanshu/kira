package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func enter() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyEnter} }

func typeInto(m *model, s string) {
	for _, r := range s {
		m.barRoute(key(string(r)))
	}
}

func TestTokenizeRespectsQuotes(t *testing.T) {
	got := tokenize(`edit KIRA-2 --title "order book merge" 'p3'`)
	want := []string{"edit", "KIRA-2", "--title", "order book merge", "p3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tokenize = %#v, want %#v", got, want)
	}
}

func TestBarClosedIgnoresOtherKeys(t *testing.T) {
	m := newTestModel(100, 12, true)
	if _, done := m.barRoute(key("j")); done {
		t.Fatal("closed bar must not swallow navigation keys")
	}
}

func TestCommandBarForwardsTokenizedArgvAndRefreshes(t *testing.T) {
	m := newTestModel(100, 12, true)
	var got []string
	m.bar.run = func(argv []string) (string, error) {
		got = argv
		return "Moved KIRA-100: OPEN -> DONE\n", nil
	}

	if _, done := m.barRoute(key(":")); !done {
		t.Fatal("':' should open the command bar")
	}
	if m.bar.mode != barCommand {
		t.Fatalf("mode = %v, want barCommand", m.bar.mode)
	}
	typeInto(&m, "move . DONE")

	cmd, done := m.barRoute(enter())
	if !done {
		t.Fatal("enter should be handled by the bar")
	}
	if m.bar.mode != barClosed {
		t.Fatal("bar should close after submit")
	}
	if cmd == nil {
		t.Fatal("command submit should return a cmd (runs off the update loop)")
	}

	res, ok := cmd().(commandResultMsg)
	if !ok {
		t.Fatalf("cmd should yield commandResultMsg, got %T", cmd())
	}
	want := []string{"move", "KIRA-100", "DONE"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argv = %#v, want %#v (focused '.' should resolve to KIRA-100)", got, want)
	}
	if !res.refresh {
		t.Fatal("a successful command should request a refresh")
	}
	if !strings.Contains(res.text, "Moved") {
		t.Fatalf("result text = %q, want command output", res.text)
	}
}

func TestCommandBarShowsErrorAndSkipsRefresh(t *testing.T) {
	m := newTestModel(100, 12, true)
	m.bar.run = func([]string) (string, error) { return "", errStub("no such state") }
	m.barRoute(key(":"))
	typeInto(&m, "move . NOPE")
	cmd, _ := m.barRoute(enter())
	if cmd == nil {
		t.Fatal("command should run even when it will fail")
	}
	res, ok := cmd().(commandResultMsg)
	if !ok {
		t.Fatalf("cmd should yield commandResultMsg, got %T", cmd())
	}
	if res.refresh {
		t.Fatal("a failed command must not refresh")
	}
	if !strings.Contains(res.text, "no such state") {
		t.Fatalf("result text = %q, want error text", res.text)
	}
}

func TestCommandResultTriggersRefresh(t *testing.T) {
	m := newTestModel(100, 12, true)
	updated, cmd := m.Update(commandResultMsg{text: "Moved KIRA-100", refresh: true})
	if got := updated.(model).bar.msg; got != "Moved KIRA-100" {
		t.Fatalf("bar msg = %q after result", got)
	}
	if cmd == nil {
		t.Fatal("a refreshing result should issue a refresh cmd")
	}
}

func TestCommandBarRejectsNestedTUI(t *testing.T) {
	m := newTestModel(100, 12, true)
	called := false
	m.bar.run = func([]string) (string, error) { called = true; return "", nil }
	m.barRoute(key(":"))
	typeInto(&m, "tui")
	if cmd, _ := m.barRoute(enter()); cmd != nil {
		cmd()
	}
	if called {
		t.Fatal("command bar must not launch a nested tui")
	}
	if !strings.Contains(m.bar.msg, "cannot launch the tui") {
		t.Fatalf("bar msg = %q, want rejection notice", m.bar.msg)
	}
}

func TestBarEscCloses(t *testing.T) {
	m := newTestModel(100, 12, true)
	m.barRoute(key(":"))
	typeInto(&m, "move")
	if _, done := m.barRoute(tea.KeyMsg{Type: tea.KeyEsc}); !done {
		t.Fatal("esc should be handled by an open bar")
	}
	if m.bar.mode != barClosed {
		t.Fatal("esc should close the bar")
	}
}

func TestHelpIncludesCommandBarKeys(t *testing.T) {
	m := newTestModel(100, 12, true)
	help := helpBody(m.activeKeys())
	for _, want := range []string{"command", "filter"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help body missing %q:\n%s", want, help)
		}
	}
}

type errStub string

func (e errStub) Error() string { return string(e) }
