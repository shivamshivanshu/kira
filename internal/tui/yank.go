package tui

import (
	"strconv"
	"strings"

	"github.com/shivamshivanshu/kira/internal/showfmt"
)

type yankEntry struct {
	label   string
	preview string
}

type yankPicker struct {
	number  string
	cursor  int
	entries []yankEntry
}

func selectedItem(m *model) (showfmt.Item, bool) {
	ts, ok := m.current().(*treeScreen)
	if !ok {
		return showfmt.Item{}, false
	}
	cur := ts.tree.current()
	if cur == nil {
		return showfmt.Item{}, false
	}
	return showfmt.Item{ID: cur.node.ID, Number: cur.node.Number, Title: cur.node.Title}, true
}

func (m *model) yankSelected() {
	if it, ok := selectedItem(m); ok {
		if text, err := showfmt.Format(showfmt.FormID, it); err == nil {
			_ = m.clip.Copy(text)
		}
	}
}

func (m *model) openYankPicker() {
	it, ok := selectedItem(m)
	if !ok {
		return
	}
	entries := make([]yankEntry, 0, len(showfmt.Forms))
	for _, f := range showfmt.Forms {
		preview, err := showfmt.Format(f, it)
		if err != nil {
			continue
		}
		entries = append(entries, yankEntry{label: f.Label(), preview: preview})
	}
	m.yank = &yankPicker{number: it.Number, entries: entries}
}

func (m *model) updateYank(key string) {
	switch key {
	case "j", "down":
		m.yank.cursor = clamp(m.yank.cursor+1, 0, len(m.yank.entries)-1)
	case "k", "up":
		m.yank.cursor = clamp(m.yank.cursor-1, 0, len(m.yank.entries)-1)
	case "enter":
		m.commitYank(m.yank.cursor)
	case "q", "esc":
		m.yank = nil
	default:
		if n, err := strconv.Atoi(key); err == nil && n >= 1 && n <= len(m.yank.entries) {
			m.commitYank(n - 1)
		}
	}
}

func (m *model) commitYank(i int) {
	if i >= 0 && i < len(m.yank.entries) {
		_ = m.clip.Copy(m.yank.entries[i].preview)
	}
	m.yank = nil
}

func (m model) renderYankPicker(height int) string {
	t := m.theme
	lines := []string{t.Accent.Bold(true).Render("yank " + m.yank.number)}
	for i, e := range m.yank.entries {
		marker, label := "  ", e.label
		if i == m.yank.cursor {
			marker, label = "> ", t.Accent.Render(e.label)
		}
		lines = append(lines, marker+label+"  "+t.Dim.Render(e.preview))
	}
	return t.Renderer().NewStyle().Width(m.width).Height(height).Padding(1, 2).Render(strings.Join(lines, "\n"))
}
