package tui

import (
	"github.com/shivamshivanshu/kira/internal/showfmt"
)

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
	entries := make([]pickerEntry, 0, len(showfmt.Forms))
	for _, f := range showfmt.Forms {
		preview, err := showfmt.Format(f, it)
		if err != nil {
			continue
		}
		entries = append(entries, pickerEntry{label: f.Label(), detail: preview, value: preview})
	}
	m.yank = &picker{title: "yank " + it.Number, entries: entries}
}

func (m *model) updateYank(key string) {
	commit, done := m.yank.update(key)
	if !done {
		return
	}
	if commit >= 0 {
		_ = m.clip.Copy(m.yank.entries[commit].value)
	}
	m.yank = nil
}
