package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func deref(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func styleText(s lipgloss.Style, text string, bold bool) string {
	if bold {
		s = s.Bold(true)
	}
	return s.Render(text)
}

func fitWidth(s string, budget int) string {
	if budget <= 0 {
		return ""
	}
	return ansi.Truncate(s, budget, "…")
}
