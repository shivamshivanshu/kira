package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
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
	{"enter", "detail"},
	{"p", "peek"},
}

func init() { registerScreen(viewBoard, func() screen { return newBoardScreen() }) }

type boardScreen struct {
	board       boardModel
	panel       *detailPanel
	detail      *datamodel.ShowResult
	detailCache map[string]*datamodel.ShowResult
	notice      string
	loaded      bool
	peek        peekMode
}

func newBoardScreen() *boardScreen {
	return &boardScreen{panel: newDetailPanel(), detailCache: map[string]*datamodel.ShowResult{}}
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

func (s *boardScreen) ensureLoaded(m *model) {
	if s.loaded || m.store == nil {
		return
	}
	s.reload(m)
	s.loaded = true
}

func (s *boardScreen) reload(m *model) {
	res, err := m.store.Board(m.cfg, core.BoardOpts{})
	if err != nil {
		s.notice = err.Error()
		return
	}
	s.board.load(res)
	s.detailCache = map[string]*datamodel.ShowResult{}
}

func (s *boardScreen) update(m *model, key string) tea.Cmd {
	s.ensureLoaded(m)
	if s.peek == peekOverlay {
		return s.panel.update(m, s.detail, key)
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
		s.moveCard(m, -1)
	case "L":
		s.moveCard(m, 1)
	case "tab", "shift+tab":
		if s.peek == peekDocked {
			s.peek = peekOverlay
			s.panel.reset()
		}
	case "enter":
		s.syncDetail(m)
		s.panel.reset()
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
		s.panel.reset()
	}
}

func (s *boardScreen) moveCard(m *model, dir int) {
	s.notice = ""
	card, ok := s.board.selected()
	if !ok {
		return
	}
	cols := s.board.columns()
	target := s.board.col + dir
	if target < 0 || target >= len(cols) {
		return
	}
	to := cols[target].State
	if m.cfg == nil || !core.AdjacentAllowed(m.cfg, s.board.result.Type, card.State, to) {
		s.notice = card.Number + ": " + card.State + " -> " + to + " is not an allowed transition"
		return
	}
	if m.store == nil {
		return
	}
	res, err := m.store.Move(m.cfg, card.ID, to, core.MoveOpts{})
	if err != nil {
		s.notice = err.Error()
		return
	}
	if len(res.Warnings) > 0 {
		s.notice = strings.Join(res.Warnings, "; ")
	} else {
		s.notice = "Moved " + res.Number + ": " + res.From + " -> " + res.To
	}
	s.reload(m)
	s.board.focusByID(card.ID)
	s.syncPeek(m)
}

func (s *boardScreen) syncDetail(m *model) {
	card, ok := s.board.selected()
	if !ok {
		s.detail = nil
		return
	}
	if cached, ok := s.detailCache[card.ID]; ok {
		s.detail = cached
		return
	}
	if m.store == nil {
		s.detail = nil
		return
	}
	res, err := loadDetail(m.store, m.cfg, card.ID)
	if err != nil {
		s.detail = nil
		return
	}
	s.detailCache[card.ID] = res
	s.detail = res
}

func (s *boardScreen) back(m *model) bool {
	if s.peek != peekOff {
		s.peek = peekOff
		return true
	}
	m.view = viewTree
	return true
}

func (s *boardScreen) focusItem(m *model, id string) {
	s.ensureLoaded(m)
	s.board.focusByID(id)
}

func (s *boardScreen) view(m *model, width, height int) string {
	s.ensureLoaded(m)
	body := height
	if s.notice != "" {
		body = height - 1
	}
	main := s.renderMain(m, width, body)
	if s.notice != "" {
		return main + "\n" + m.theme.Dim.Render(m.theme.Renderer().NewStyle().MaxWidth(width).Render(s.notice))
	}
	return main
}

func (s *boardScreen) renderMain(m *model, width, height int) string {
	if s.peek == peekOverlay || (s.peek == peekDocked && !splitDetail(width)) {
		return s.panel.render(m.theme, s.detail, width, height)
	}
	if s.peek == peekDocked {
		bw := treeWidth(width)
		left := renderBoard(m.theme, m.icons, s.board.result, bw, height, s.board.col, s.board.row)
		sep := verticalRule(m.theme.Border.Render("│"), height)
		right := s.panel.render(m.theme, s.detail, width-bw-1, height)
		return lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
	}
	return renderBoard(m.theme, m.icons, s.board.result, width, height, s.board.col, s.board.row)
}
