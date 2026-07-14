package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

const MinWidth = 80

func splitDetail(width int) bool { return width >= MinWidth }

func treeWidth(width int, split float64) int {
	if !splitDetail(width) {
		return width
	}
	return max(int(float64(width)*clampSplit(split)), MinWidth/2)
}

func clampSplit(split float64) float64 {
	if split <= 0 || split >= 1 {
		return datamodel.DefaultSplit
	}
	return split
}

func splitPane(t theme.Theme, split float64, width, height int, left, right func(w int) string) string {
	lw := treeWidth(width, split)
	sep := verticalRule(t.Border.Render("│"), height)
	return lipgloss.JoinHorizontal(lipgloss.Top, left(lw), sep, right(width-lw-1))
}
