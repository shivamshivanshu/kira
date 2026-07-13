package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

func frameOf(t theme.Theme, width, height int) lipgloss.Style {
	return t.Renderer().NewStyle().Width(width).MaxHeight(height)
}

func renderScrollable(t theme.Theme, lines []string, scroll *int, width, height int) string {
	*scroll = clamp(*scroll, 0, max(0, len(lines)-height))
	end := min(len(lines), *scroll+height)
	return frameOf(t, width, height).Render(strings.Join(lines[*scroll:end], "\n"))
}
