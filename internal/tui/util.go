package tui

import "github.com/charmbracelet/x/ansi"

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

func fitWidth(s string, budget int) string {
	if budget <= 0 {
		return ""
	}
	return ansi.Truncate(s, budget, "…")
}
