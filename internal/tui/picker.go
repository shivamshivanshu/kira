package tui

import (
	"strconv"
	"strings"

	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

type pickerEntry struct {
	label  string
	detail string
	value  string
}

type picker struct {
	title   string
	cursor  int
	entries []pickerEntry
}

func (p *picker) update(key string) (commit int, done bool) {
	switch key {
	case "j", "down":
		p.cursor = clamp(p.cursor+1, 0, len(p.entries)-1)
	case "k", "up":
		p.cursor = clamp(p.cursor-1, 0, len(p.entries)-1)
	case "enter":
		return p.cursor, true
	case "q", "esc":
		return -1, true
	default:
		if n, err := strconv.Atoi(key); err == nil && n >= 1 && n <= len(p.entries) {
			return n - 1, true
		}
	}
	return -1, false
}

func (p *picker) render(t theme.Theme, width, height int) string {
	lines := []string{t.Accent.Bold(true).Render(p.title)}
	for i, e := range p.entries {
		marker, label := "  ", e.label
		if i == p.cursor {
			marker, label = "> ", t.Accent.Render(e.label)
		}
		line := marker + label
		if e.detail != "" {
			line += "  " + t.Dim.Render(e.detail)
		}
		lines = append(lines, line)
	}
	return t.Renderer().NewStyle().Width(width).Height(height).Padding(1, 2).Render(strings.Join(lines, "\n"))
}
