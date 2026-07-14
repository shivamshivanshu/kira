package tui

import (
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func (m *model) openBoardScope(current string) {
	entries := []pickerEntry{{label: "All boards", value: ""}}
	if m.cfg != nil {
		for _, b := range m.cfg.ActiveBoards() {
			entries = append(entries, pickerEntry{label: boardScopeLabel(b), value: b.Key})
		}
	}
	cursor := 0
	for i, e := range entries {
		if strings.EqualFold(e.value, current) {
			cursor = i
			break
		}
	}
	m.picker = &picker{
		title:   "board scope",
		cursor:  cursor,
		entries: entries,
		onCommit: func(m *model, entry pickerEntry) {
			if bs, ok := m.boardScreen(); ok {
				bs.scope = entry.value
				bs.applyScope()
			}
		},
	}
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
