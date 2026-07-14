package tui

import (
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/shivamshivanshu/kira/internal/clipx"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

type view int

const (
	viewTree view = iota
	viewBoard
	viewStats
)

var viewOrder = []view{viewTree, viewBoard, viewStats}

var viewLabel = map[view]string{viewTree: "tree", viewBoard: "board", viewStats: "stats"}

var globalKeys = []KeyBinding{
	{"1/2/3", "view"},
	{"^o/^]", "jump"},
	{":", "command"},
	{"/", "filter"},
	{"y/Y", "yank"},
	{"r", "refresh"},
	{"?", "help"},
	{"q", "quit"},
}

type model struct {
	store *core.Store
	cfg   *datamodel.Config
	theme theme.Theme
	icons iconSet

	width   int
	height  int
	help    bool
	jumps   jumplist
	loadErr error

	view      view
	screens   map[view]screen
	bar       bar
	clip      clipx.Clipboard
	yank      *yankPicker
	boardPick *boardPicker

	crash       *crashInfo
	injectPanic bool

	busy         bool
	quitting     bool
	pending      []tea.Cmd
	refreshEvery time.Duration
}

func newModel(store *core.Store, cfg *datamodel.Config, th theme.Theme, ic iconSet, injectPanic bool) model {
	var every time.Duration
	if cfg != nil {
		every = cfg.UI.Tui.RefreshInterval()
	}
	return model{store: store, cfg: cfg, theme: th, icons: ic, view: viewTree, screens: buildScreens(), bar: newBar(), injectPanic: injectPanic, refreshEvery: every}
}

func (m model) Init() tea.Cmd {
	if m.injectPanic {
		return safeCmd(func() tea.Msg { panic("injected tui panic (tea.Cmd)") })
	}
	return m.scheduleRefresh()
}

func (m model) scheduleRefresh() tea.Cmd {
	if m.refreshEvery <= 0 {
		return nil
	}
	return tea.Tick(m.refreshEvery, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case crashMsg:
		m.crash = &crashInfo{value: msg.value, stack: msg.stack}
		return m, tea.Quit
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case treeLoadedMsg:
		cmd := m.drain()
		if msg.err != nil {
			m.loadErr = msg.err
			return m, m.afterDrain(cmd)
		}
		m.loadErr = nil
		if ts, ok := m.screens[viewTree].(*treeScreen); ok {
			ts.setData(&m, msg.data)
		}
		if bs, ok := m.screens[viewBoard].(*boardScreen); ok {
			bs.invalidate()
		}
		if ss, ok := m.screens[viewStats].(*statsScreen); ok {
			ss.invalidate()
		}
		return m, m.afterDrain(cmd)
	case commandResultMsg:
		m.bar.msg = msg.text
		m.bar.msgErr = msg.isError
		if msg.refresh {
			m.pending = append(m.pending, refreshCmd(m.store, m.cfg, m.bar.filter))
		}
		return m, m.afterDrain(m.drain())
	case boardMovedMsg:
		cmd := m.drain()
		if bs, ok := m.screens[viewBoard].(*boardScreen); ok {
			bs.applyMove(&m, msg)
		}
		return m, m.afterDrain(cmd)
	case tickMsg:
		var cmd tea.Cmd
		if !m.busy {
			cmd = m.request(refreshCmd(m.store, m.cfg, m.bar.filter))
		}
		return m, tea.Batch(cmd, m.scheduleRefresh())
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		return m, m.requestQuit(true)
	}
	if cmd, done := m.barRoute(msg); done {
		return m, cmd
	}
	if m.yank != nil {
		m.updateYank(key)
		return m, nil
	}
	if m.boardPick != nil {
		m.updateBoardScope(key)
		return m, nil
	}
	switch key {
	case "y":
		m.yankSelected()
		return m, nil
	case "Y":
		m.openYankPicker()
		return m, nil
	case "q", "esc":
		if m.help {
			m.help = false
			return m, nil
		}
		if s := m.current(); s != nil && s.back(&m) {
			return m, nil
		}
		return m, m.requestQuit(false)
	case "?":
		m.help = !m.help
		return m, nil
	case "1":
		m.switchView(viewTree)
		return m, nil
	case "2":
		m.switchView(viewBoard)
		return m, nil
	case "3":
		m.switchView(viewStats)
		return m, nil
	case "r":
		return m, m.request(refreshCmd(m.store, m.cfg, m.bar.filter))
	case "ctrl+o":
		m.jumpTo(m.jumps.back())
		return m, nil
	case "ctrl+]":
		m.jumpTo(m.jumps.forward())
		return m, nil
	default:
		s := m.current()
		if s == nil {
			return m, nil
		}
		cmd := s.update(&m, key)
		return m, cmd
	}
}

func (m *model) request(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	m.pending = append(m.pending, cmd)
	if m.busy {
		return nil
	}
	return m.drain()
}

func (m *model) afterDrain(next tea.Cmd) tea.Cmd {
	if next != nil {
		return next
	}
	if m.quitting {
		return tea.Quit
	}
	if s := m.current(); s != nil {
		s.settle(m)
	}
	return nil
}

func (m *model) requestQuit(force bool) tea.Cmd {
	if !m.busy {
		return tea.Quit
	}
	if m.quitting && force {
		return tea.Quit
	}
	m.quitting = true
	return nil
}

func (m *model) drain() tea.Cmd {
	if len(m.pending) == 0 {
		m.busy = false
		return nil
	}
	next := m.pending[0]
	m.pending = m.pending[1:]
	m.busy = true
	return next
}

func (m *model) current() screen { return m.screens[m.view] }

func (m *model) switchView(v view) {
	if s := m.current(); s != nil {
		m.jumps.push(jumpEntry{view: m.view, itemID: focusedItem(s)})
	}
	m.view = v
}

func (m *model) jumpTo(e jumpEntry, ok bool) {
	if !ok {
		return
	}
	m.view = e.view
	if s := m.current(); s != nil {
		s.focusItem(m, e.itemID)
	}
}

func (m model) mainHeight() int { return max(1, m.height-2) }

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	h := m.mainHeight()
	var main string
	switch {
	case m.yank != nil:
		main = m.renderYankPicker(h)
	case m.boardPick != nil:
		main = m.renderBoardScope(h)
	case m.help:
		main = m.renderHelp(h)
	case m.current() != nil:
		main = m.current().view(&m, m.width, h)
	default:
		main = m.renderMissing(h)
	}
	return m.renderTitle() + "\n" + main + "\n" + m.footer()
}

func (m model) renderTitle() string {
	parts := make([]string, 0, len(viewOrder)+1)
	parts = append(parts, m.theme.Accent.Bold(true).Render("kira"))
	for i, v := range viewOrder {
		label := strconv.Itoa(i+1) + ":" + viewLabel[v]
		if v == m.view {
			parts = append(parts, m.theme.Accent.Bold(true).Render(label))
		} else {
			parts = append(parts, m.theme.Dim.Render(label))
		}
	}
	return m.theme.Renderer().NewStyle().Width(m.width).MaxWidth(m.width).Render(strings.Join(parts, "  "))
}

func (m model) activeKeys() []KeyBinding {
	if s := m.current(); s != nil {
		return append(append([]KeyBinding{}, s.keys()...), globalKeys...)
	}
	return globalKeys
}

func (m model) renderHint() string {
	fit := m.theme.Renderer().NewStyle().MaxWidth(m.width)
	if m.loadErr != nil {
		return m.theme.Heat.Hot.Render(fit.Render("load failed: " + m.loadErr.Error()))
	}
	return m.theme.Dim.Render(fit.Render(hintLine(m.activeKeys())))
}

func (m model) renderHelp(h int) string {
	body := m.theme.Text.Render(helpBody(m.activeKeys()))
	return m.theme.Renderer().NewStyle().Width(m.width).Height(h).Padding(1, 2).Render(body)
}

func (m model) renderMissing(h int) string {
	name := viewLabel[m.view]
	if name != "" {
		name = strings.ToUpper(name[:1]) + name[1:]
	}
	msg := m.theme.Dim.Render(name + " is not available yet — press 1 for the tree")
	return centered(m.theme, m.width, h, msg)
}

func centered(t theme.Theme, width, height int, s string) string {
	return t.Renderer().NewStyle().Width(width).Height(height).Align(lipgloss.Center, lipgloss.Center).Render(s)
}

func focusedItem(s screen) string {
	if ts, ok := s.(*treeScreen); ok {
		return ts.tree.selectedID()
	}
	return ""
}
