package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

const MinWidth = 80

func splitDetail(width int) bool { return width >= MinWidth }

func treeWidth(width int) int {
	if !splitDetail(width) {
		return width
	}
	return max(width/2, MinWidth/2)
}

func splitPane(t theme.Theme, width, height int, left, right func(w int) string) string {
	lw := treeWidth(width)
	sep := verticalRule(t.Border.Render("│"), height)
	return lipgloss.JoinHorizontal(lipgloss.Top, left(lw), sep, right(width-lw-1))
}
