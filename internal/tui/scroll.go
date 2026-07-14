package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

const scrollToEnd = 1 << 30

type scroller struct {
	scroll   int
	pendingG bool
}

func (s *scroller) update(key string, halfPage int) bool {
	if s.pendingG {
		s.pendingG = false
		if key == "g" {
			s.scroll = 0
		}
		return true
	}
	switch key {
	case "j", "down":
		s.scroll++
	case "k", "up":
		s.scroll--
	case "ctrl+d":
		s.scroll += halfPage
	case "ctrl+u":
		s.scroll -= halfPage
	case "g":
		s.pendingG = true
	case "G":
		s.scroll = scrollToEnd
	default:
		return false
	}
	return true
}

func frameOf(t theme.Theme, width, height int) lipgloss.Style {
	return t.Renderer().NewStyle().Width(width).MaxHeight(height)
}

func renderScrollable(t theme.Theme, lines []string, scroll *int, width, height int) string {
	*scroll = clamp(*scroll, 0, max(0, len(lines)-height))
	end := min(len(lines), *scroll+height)
	return frameOf(t, width, height).Render(strings.Join(lines[*scroll:end], "\n"))
}
