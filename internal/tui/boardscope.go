package tui

import (
	"strconv"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

type boardEntry struct {
	key   string
	label string
}

type boardPicker struct {
	cursor  int
	entries []boardEntry
}

func (m *model) openBoardScope(current string) {
	entries := []boardEntry{{key: "", label: "All boards"}}
	if m.cfg != nil {
		for _, b := range m.cfg.ActiveBoards() {
			entries = append(entries, boardEntry{key: b.Key, label: boardScopeLabel(b)})
		}
	}
	cursor := 0
	for i, e := range entries {
		if strings.EqualFold(e.key, current) {
			cursor = i
			break
		}
	}
	m.boardPick = &boardPicker{cursor: cursor, entries: entries}
}

func (m *model) updateBoardScope(key string) {
	switch key {
	case "j", "down":
		m.boardPick.cursor = clamp(m.boardPick.cursor+1, 0, len(m.boardPick.entries)-1)
	case "k", "up":
		m.boardPick.cursor = clamp(m.boardPick.cursor-1, 0, len(m.boardPick.entries)-1)
	case "enter":
		m.commitBoardScope(m.boardPick.cursor)
	case "q", "esc":
		m.boardPick = nil
	default:
		if n, err := strconv.Atoi(key); err == nil && n >= 1 && n <= len(m.boardPick.entries) {
			m.commitBoardScope(n - 1)
		}
	}
}

func (m *model) commitBoardScope(i int) {
	if i >= 0 && i < len(m.boardPick.entries) {
		if bs, ok := m.screens[viewBoard].(*boardScreen); ok {
			bs.scope = m.boardPick.entries[i].key
			bs.applyScope()
		}
	}
	m.boardPick = nil
}

func (m model) renderBoardScope(height int) string {
	t := m.theme
	lines := []string{t.Accent.Bold(true).Render("board scope")}
	for i, e := range m.boardPick.entries {
		marker, label := "  ", e.label
		if i == m.boardPick.cursor {
			marker, label = "> ", t.Accent.Render(e.label)
		}
		lines = append(lines, marker+label)
	}
	return t.Renderer().NewStyle().Width(m.width).Height(height).Padding(1, 2).Render(strings.Join(lines, "\n"))
}

func boardScopeLabel(b datamodel.Board) string {
	if b.Name != "" && !strings.EqualFold(b.Name, b.Key) {
		return b.Key + "  " + b.Name
	}
	return b.Key
}

func scopedBoard(res *datamodel.BoardResult, scope string) *datamodel.BoardResult {
	if res == nil || scope == "" {
		return res
	}
	cols := make([]datamodel.BoardColumn, len(res.Columns))
	for i, c := range res.Columns {
		c.Items = filterByBoard(c.Items, scope)
		cols[i] = c
	}
	return &datamodel.BoardResult{Type: res.Type, Columns: cols}
}

func filterByBoard(items []datamodel.ListItem, scope string) []datamodel.ListItem {
	out := make([]datamodel.ListItem, 0, len(items))
	for _, it := range items {
		if strings.EqualFold(it.Board, scope) {
			out = append(out, it)
		}
	}
	return out
}
