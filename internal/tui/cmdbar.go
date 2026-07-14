package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type barMode int

const (
	barClosed barMode = iota
	barCommand
	barFilter
)

type bar struct {
	mode   barMode
	input  textinput.Model
	msg    string
	msgErr bool
	filter string
	run    func([]string) (string, error)
}

type commandResultMsg struct {
	text    string
	refresh bool
	isError bool
}

func newBar() bar { return bar{input: textinput.New()} }

func (m *model) barRoute(msg tea.KeyMsg) (tea.Cmd, bool) {
	if m.bar.mode == barClosed {
		switch msg.String() {
		case ":":
			m.bar.open(barCommand, "")
			return textinput.Blink, true
		case "/":
			m.bar.open(barFilter, m.bar.filter)
			return textinput.Blink, true
		}
		return nil, false
	}
	switch msg.String() {
	case "ctrl+c":
		return nil, false
	case "esc":
		m.bar.close()
		return nil, true
	case "enter":
		return m.barSubmit(), true
	}
	var cmd tea.Cmd
	m.bar.input, cmd = m.bar.input.Update(msg)
	return cmd, true
}

func (b *bar) open(mode barMode, value string) {
	b.mode = mode
	b.msg = ""
	b.msgErr = false
	if mode == barFilter {
		b.input.Prompt = "/"
	} else {
		b.input.Prompt = ":"
	}
	b.input.SetValue(value)
	b.input.CursorEnd()
	b.input.Focus()
}

func (b *bar) close() {
	b.mode = barClosed
	b.input.Blur()
	b.input.SetValue("")
}

func (b *bar) setError(text string) {
	b.msg = text
	b.msgErr = true
}

func (m *model) barSubmit() tea.Cmd {
	value := strings.TrimSpace(m.bar.input.Value())
	switch m.bar.mode {
	case barCommand:
		return m.request(m.runCommand(value))
	case barFilter:
		return m.applyFilter(value)
	}
	return nil
}

func (m *model) runCommand(line string) tea.Cmd {
	m.bar.close()
	argv := m.substituteFocused(tokenize(line))
	if len(argv) == 0 {
		return nil
	}
	if argv[0] == "tui" {
		m.bar.setError("cannot launch the tui from the command bar")
		return nil
	}
	run := m.bar.run
	if run == nil {
		m.bar.setError("command bar unavailable")
		return nil
	}
	return safeCmd(func() tea.Msg {
		out, err := run(argv)
		if err != nil {
			return commandResultMsg{text: firstNonEmptyLine(err.Error()), isError: true}
		}
		return commandResultMsg{text: firstNonEmptyLine(out), refresh: true}
	})
}

func (m *model) applyFilter(expr string) tea.Cmd {
	m.bar.close()
	if m.store == nil {
		m.bar.setError("filter unavailable")
		return nil
	}
	m.bar.filter = expr
	return m.request(refreshCmd(m.store, m.cfg, expr))
}

func (m *model) substituteFocused(argv []string) []string {
	number := m.focusedNumber()
	if number == "" {
		return argv
	}
	out := make([]string, len(argv))
	for i, tok := range argv {
		if tok == "." {
			out[i] = number
		} else {
			out[i] = tok
		}
	}
	return out
}

func (m *model) focusedNumber() string {
	if it, ok := selectedItem(m); ok {
		return it.Number
	}
	return ""
}

func (m model) footer() string {
	style := m.theme.Renderer().NewStyle().MaxWidth(m.width)
	switch {
	case m.quitting:
		return m.theme.Heat.Warm.Render(style.Render("finishing in-flight change… (ctrl+c to force quit)"))
	case m.bar.mode != barClosed:
		return style.Render(m.bar.input.View())
	case m.bar.msg != "":
		msgStyle := m.theme.Dim
		if m.bar.msgErr {
			msgStyle = m.theme.Heat.Hot
		}
		return msgStyle.Render(style.Render(m.bar.msg))
	case m.bar.filter != "":
		return m.theme.Dim.Render(style.Render("filter: " + m.bar.filter + "  (/ change · : command)"))
	default:
		return m.renderHint()
	}
}

func tokenize(line string) []string {
	var tokens []string
	var cur strings.Builder
	inWord := false
	quote := rune(0)
	flush := func() {
		if inWord {
			tokens = append(tokens, cur.String())
			cur.Reset()
			inWord = false
		}
	}
	for _, r := range line {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			inWord = true
		case r == ' ' || r == '\t':
			flush()
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	flush()
	return tokens
}

func firstNonEmptyLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return "ok"
}
