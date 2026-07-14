package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shivamshivanshu/kira/internal/core"
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
	board  boardModel
	host   detailHost
	notice string
	loaded bool
	peek   peekMode
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
		s.notice = err.Error()
		return
	}
	s.board.load(res)
	s.host.resetCache()
}

func (s *boardScreen) update(m *model, key string) tea.Cmd {
	s.ensureLoaded(m)
	if s.peek == peekOverlay {
		return s.host.update(m, key)
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
	s.notice = ""
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
		s.notice = card.Number + ": " + card.State + " -> " + to + " is not an allowed transition"
		return nil
	}
	if m.store == nil {
		return nil
	}
	return m.request(boardMoveCmd(m.store, m.cfg, card.ID, to))
}

func (s *boardScreen) applyMove(m *model, msg boardMovedMsg) {
	if msg.res == nil {
		s.notice = msg.err.Error()
		return
	}
	if len(msg.res.Warnings) > 0 {
		s.notice = strings.Join(msg.res.Warnings, "; ")
	} else {
		s.notice = "Moved " + msg.res.Number + ": " + msg.res.From + " -> " + msg.res.To
	}
	if msg.err != nil || msg.board == nil {
		s.loaded = false
		return
	}
	s.board.load(msg.board)
	s.host.resetCache()
	s.board.focusByID(msg.cardID)
	s.syncPeek(m)
}

func (s *boardScreen) settle(m *model) { s.host.settle(m) }

func (s *boardScreen) syncDetail(m *model) {
	card, _ := s.board.selected()
	s.host.sync(m, card.ID)
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
