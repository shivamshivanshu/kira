package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/showfmt"
)

type peekMode int

const (
	peekOff peekMode = iota
	peekDocked
	peekOverlay
)

var boardKeys = []KeyBinding{
	{"h/l", "column"},
	{"j/k", "card"},
	{"H/L", "move"},
	{"b", "board"},
	{"enter", "detail"},
	{"p", "peek"},
	{"gg/G", "top/bottom"},
}

type boardScreen struct {
	board     boardModel
	host      detailHost
	raw       *datamodel.BoardResult
	scope     string
	notice    string
	noticeErr bool
	loaded    bool
	peek      peekMode
	chord     chord
}

func newBoardScreen() *boardScreen {
	return &boardScreen{host: newDetailHost()}
}

func (s *boardScreen) keys() []KeyBinding {
	switch s.peek {
	case peekOverlay:
		return detailKeys
	case peekDocked:
		return append(append([]KeyBinding{}, boardKeys...), KeyBinding{"tab", "expand"})
	default:
		return boardKeys
	}
}

func (s *boardScreen) invalidate() { s.loaded = false }

func (s *boardScreen) ensureLoaded(m *model) {
	if s.loaded || m.store == nil || m.busy {
		return
	}
	s.reload(m)
	s.loaded = true
}

func (s *boardScreen) reload(m *model) {
	res, err := m.store.Board(m.cfg, core.BoardOpts{})
	if err != nil {
		s.setError(err.Error())
		return
	}
	s.raw = res
	s.applyScope()
}

func (s *boardScreen) setNotice(text string) {
	s.notice, s.noticeErr = text, false
}

func (s *boardScreen) setError(text string) {
	s.notice, s.noticeErr = firstNonEmptyLine(text), true
}

func (s *boardScreen) applyScope() {
	s.board.load(scopedBoard(s.raw, s.scope))
	s.host.resetCache()
}

func (s *boardScreen) update(m *model, key string) tea.Cmd {
	s.ensureLoaded(m)
	if s.peek == peekOverlay {
		cmd, _ := s.host.update(m, key)
		return cmd
	}
	if p, ok := s.chord.take(); ok {
		if p+key == "gg" {
			s.board.toTop()
			s.syncPeek(m)
		}
		return nil
	}
	switch key {
	case "j", "down":
		s.board.moveRow(1)
		s.syncPeek(m)
	case "k", "up":
		s.board.moveRow(-1)
		s.syncPeek(m)
	case "l", "right":
		s.board.moveCol(1)
		s.syncPeek(m)
	case "h", "left":
		s.board.moveCol(-1)
		s.syncPeek(m)
	case "H":
		return s.moveCard(m, -1)
	case "L":
		return s.moveCard(m, 1)
	case "b":
		m.openBoardScope(s.scope)
	case "g":
		s.chord.arm(key)
	case "G":
		s.board.toBottom()
		s.syncPeek(m)
	case "tab", "shift+tab":
		if s.peek == peekDocked {
			s.peek = peekOverlay
			s.host.panel.reset()
		}
	case "enter":
		s.syncDetail(m)
		s.host.panel.reset()
		s.peek = peekOverlay
	case "p":
		if s.peek == peekOff {
			s.peek = peekDocked
		} else {
			s.peek = peekOff
		}
		s.syncPeek(m)
	}
	return nil
}

func (s *boardScreen) syncPeek(m *model) {
	if s.peek != peekOff {
		s.syncDetail(m)
		s.host.panel.reset()
	}
}

func (s *boardScreen) moveCard(m *model, dir int) tea.Cmd {
	s.setNotice("")
	card, ok := s.board.selected()
	if !ok {
		return nil
	}
	cols := s.board.columns()
	target := s.board.col + dir
	if target < 0 || target >= len(cols) {
		return nil
	}
	to := cols[target].State
	if m.cfg == nil || !core.AdjacentAllowed(m.cfg, s.board.result.Type, card.State, to) {
		s.setNotice(card.Number + ": " + card.State + " -> " + to + " is not an allowed transition")
		return nil
	}
	if m.store == nil {
		return nil
	}
	return m.request(boardMoveCmd(m.store, m.cfg, card.ID, to))
}

func (s *boardScreen) applyMove(m *model, msg boardMovedMsg) {
	if msg.res == nil {
		s.setError(msg.err.Error())
		return
	}
	if len(msg.res.Warnings) > 0 {
		s.setNotice(strings.Join(msg.res.Warnings, "; "))
	} else {
		s.setNotice("Moved " + msg.res.Number + ": " + msg.res.From + " -> " + msg.res.To)
	}
	if msg.err != nil || msg.board == nil {
		s.loaded = false
		return
	}
	s.raw = msg.board
	s.applyScope()
	s.board.focusByID(msg.cardID)
	s.syncPeek(m)
}

func (s *boardScreen) settle(m *model) { s.host.settle(m) }

func (s *boardScreen) syncDetail(m *model) {
	card, _ := s.board.selected()
	s.host.sync(m, card.ID)
}

func (s *boardScreen) back(_ *model) bool {
	if s.peek != peekOff {
		s.peek = peekOff
		return true
	}
	return false
}

func (s *boardScreen) focusItem(m *model, id string) {
	s.ensureLoaded(m)
	s.board.focusByID(id)
}

func (s *boardScreen) focusedItem() (showfmt.Item, bool) {
	card, ok := s.board.selected()
	if !ok {
		return showfmt.Item{}, false
	}
	return showfmt.Item{ID: card.ID, Number: card.Number, Title: card.Title}, true
}

func (s *boardScreen) view(m *model, width, height int) string {
	s.ensureLoaded(m)
	header := s.scopeHeader(m, width)
	body := height
	if header != "" {
		body--
	}
	if s.notice != "" {
		body--
	}
	main := s.renderMain(m, width, body)
	lines := make([]string, 0, 3)
	if header != "" {
		lines = append(lines, header)
	}
	lines = append(lines, main)
	if s.notice != "" {
		style := m.theme.Dim
		if s.noticeErr {
			style = m.theme.Heat.Hot
		}
		lines = append(lines, style.Render(m.theme.Renderer().NewStyle().MaxWidth(width).Render(s.notice)))
	}
	return strings.Join(lines, "\n")
}

func (s *boardScreen) scopeHeader(m *model, width int) string {
	if s.scope == "" {
		return ""
	}
	return m.theme.Dim.Render(m.theme.Renderer().NewStyle().MaxWidth(width).Render("board: " + s.scope))
}

func (s *boardScreen) renderMain(m *model, width, height int) string {
	if s.board.result == nil && m.busy {
		return centered(m.theme, width, height, m.theme.Dim.Render("loading…"))
	}
	if s.peek == peekOverlay || (s.peek == peekDocked && !splitDetail(width)) {
		return s.host.render(m.theme, m.icons, width, height)
	}
	if s.peek == peekDocked {
		return splitPane(m.theme, m.cfg.UI.Tui.Split, width, height,
			func(w int) string {
				return renderBoard(m.theme, m.icons, s.board.result, w, height, s.board.col, s.board.row)
			},
			func(w int) string { return s.host.render(m.theme, m.icons, w, height) })
	}
	return renderBoard(m.theme, m.icons, s.board.result, width, height, s.board.col, s.board.row)
}
