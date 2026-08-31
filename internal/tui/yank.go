package tui

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/showfmt"
)

func selectedItem(m *model) (showfmt.Item, bool) {
	if s := m.current(); s != nil {
		return s.focusedItem()
	}
	return showfmt.Item{}, false
}

func (m *model) workonSettings() datamodel.Workon {
	if m.cfg != nil {
		return m.cfg.Workon
	}
	return datamodel.Workon{}
}

func (m *model) copyToClipboard(text string) {
	if err := m.clip.Copy(text); err != nil {
		m.bar.setError("copy: " + firstNonEmptyLine(err.Error()))
	}
}

func (m *model) yankSelected() {
	if it, ok := selectedItem(m); ok {
		if text, err := showfmt.Format(showfmt.FormID, it, m.workonSettings()); err == nil {
			m.copyToClipboard(text)
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
		preview, err := showfmt.Format(f, it, m.workonSettings())
		if err != nil {
			continue
		}
		entries = append(entries, pickerEntry{label: f.Label(), detail: preview, value: preview})
	}
	m.picker = &picker{
		title:    "yank " + it.Number,
		entries:  entries,
		onCommit: func(m *model, entry pickerEntry) { m.copyToClipboard(entry.value) },
	}
}
