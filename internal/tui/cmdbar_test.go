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

func TestCommandBarRespectsQuotesAndEscapes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		line string
		want []string
	}{
		{`edit KIRA-2 --title "order book merge" 'p3'`, []string{"edit", "KIRA-2", "--title", "order book merge", "p3"}},
		{`comment . -m "say \"hi\""`, []string{"comment", "KIRA-100", "-m", `say "hi"`}},
		{`comment . -m note\ `, []string{"comment", "KIRA-100", "-m", "note "}},
	} {
		t.Run(tc.line, func(t *testing.T) {
			m := newTestModel(100, 12, true)
			var got []string
			m.bar.run = func(argv []string) (string, error) { got = argv; return "ok", nil }
			m.bar.open(barCommand, tc.line)
			cmd, _ := m.barRoute(enter())
			if cmd == nil {
				t.Fatalf("command rejected: %s", m.bar.msg)
			}
			cmd()
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("argv = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestCommandBarRejectsMalformedInput(t *testing.T) {
	t.Parallel()
	for _, line := range []string{`comment . -m "unfinished`, `edit . --title 'unfinished`, `show .\`} {
		t.Run(line, func(t *testing.T) {
			m := newTestModel(100, 12, true)
			m.bar.run = func([]string) (string, error) {
				t.Fatal("malformed command must not execute")
				return "", nil
			}
			m.bar.open(barCommand, line)
			if cmd, _ := m.barRoute(enter()); cmd != nil {
				cmd()
				t.Fatal("malformed command returned a command")
			}
			if !m.bar.msgErr || !strings.Contains(m.bar.msg, "unfinished quote or escape") {
				t.Fatalf("missing parse error: %q", m.bar.msg)
			}
		})
	}
}

func TestCommandBarEmptyInputIsNoop(t *testing.T) {
	t.Parallel()
	for _, line := range []string{"", "   "} {
		t.Run(line, func(t *testing.T) {
			m := newTestModel(100, 12, true)
			m.bar.run = func([]string) (string, error) {
				t.Fatal("empty command must not execute")
				return "", nil
			}
			m.bar.open(barCommand, line)
			if cmd, _ := m.barRoute(enter()); cmd != nil {
				cmd()
				t.Fatal("empty input returned a command")
			}
			if m.bar.msgErr || m.bar.msg != "" || m.bar.mode != barClosed {
				t.Fatalf("empty input was not a no-op: %+v", m.bar)
			}
		})
	}
}

func TestBarClosedIgnoresOtherKeys(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	if _, done := m.barRoute(key("j")); done {
		t.Fatal("closed bar must not swallow navigation keys")
	}
}

func TestNKeyPrefillsCreateInCommandBar(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	updated, cmd := m.Update(key("n"))
	m2 := updated.(model)
	if m2.bar.mode != barCommand {
		t.Fatalf("n must open the command bar, mode=%d", m2.bar.mode)
	}
	if got := m2.bar.input.Value(); got != "create " {
		t.Fatalf("n must prefill the create command, got %q", got)
	}
	if cmd == nil {
		t.Fatal("n must return the input blink command")
	}
}

func TestCommandBarForwardsTokenizedArgvAndRefreshes(t *testing.T) {
	t.Parallel()
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

func TestCommandBarDotResolvesBoardSelection(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	bs := m.screens[viewBoard].(*boardScreen)
	bs.state = loadReady
	bs.board.load(buildBoardResult())
	bs.board.focusByID("t3")
	m.view = viewBoard
	var got []string
	m.bar.run = func(argv []string) (string, error) { got = argv; return "ok", nil }

	m.barRoute(key(":"))
	typeInto(&m, "move . DONE")
	cmd, _ := m.barRoute(enter())
	cmd()
	want := []string{"move", "KIRA-160", "DONE"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argv = %#v, want %#v ('.' must resolve the board's focused card, not the tree cursor)", got, want)
	}
}

func TestCommandBarDotOnStatsStaysLiteral(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	m.view = viewStats
	var got []string
	m.bar.run = func(argv []string) (string, error) { got = argv; return "ok", nil }

	m.barRoute(key(":"))
	typeInto(&m, "show .")
	cmd, _ := m.barRoute(enter())
	cmd()
	want := []string{"show", "."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argv = %#v, want %#v (stats has no focused item to substitute)", got, want)
	}
}

func TestCommandBarShowsErrorAndSkipsRefresh(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
