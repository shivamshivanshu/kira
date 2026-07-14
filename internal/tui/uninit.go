package tui

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

type uninitModel struct {
	root  string
	theme theme.Theme
	input textinput.Model

	width  int
	height int
	msg    string

	initialized bool
}

func newUninitModel(root string, th theme.Theme) uninitModel {
	in := textinput.New()
	in.Placeholder = "project key"
	in.Prompt = "> "
	in.Focus()
	return uninitModel{root: root, theme: th, input: in}
}

func RunUninit(root string, opts Options) (bool, error) {
	out := opts.Output
	if out == nil {
		out = os.Stdout
	}
	th := theme.For(out, datamodel.UI{}, opts.NoColor)
	m := newUninitModel(root, th)

	initialized := false
	err := guardRun(root, os.Stderr, func() error {
		final, runErr := tea.NewProgram(m, programOptions(opts, out)...).Run()
		if runErr != nil {
			return runErr
		}
		if fm, ok := final.(uninitModel); ok {
			initialized = fm.initialized
		}
		return nil
	})
	return initialized, err
}

func (m uninitModel) Init() tea.Cmd { return textinput.Blink }

func (m uninitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			return m.submit()
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m uninitModel) submit() (tea.Model, tea.Cmd) {
	key := strings.TrimSpace(m.input.Value())
	if key == "" {
		m.msg = "enter a project key to create the project"
		return m, nil
	}
	if _, err := core.Init(m.root, key, false); err != nil {
		m.msg = err.Error()
		return m, nil
	}
	m.initialized = true
	return m, tea.Quit
}

func (m uninitModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(m.theme.Accent.Bold(true).Render("kira is not initialized in this directory"))
	b.WriteString("\n\n")
	b.WriteString(m.theme.Text.Render("Creating a project will:"))
	b.WriteString("\n")
	b.WriteString(m.theme.Dim.Render("  - write .kira/ (config, templates, tickets)"))
	b.WriteString("\n")
	b.WriteString(m.theme.Dim.Render(`  - make a git commit "kira: init"`))
	b.WriteString("\n\n")
	b.WriteString(m.theme.Text.Render("Enter a project key to create, or Esc to cancel."))
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	if m.msg != "" {
		b.WriteString("\n\n")
		b.WriteString(m.theme.Dim.Render(m.msg))
	}
	return m.theme.Renderer().NewStyle().Width(m.width).Height(m.height).Padding(1, 2).Render(b.String())
}
