package tui

import (
	"fmt"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

const progressBarWidth = 7

func progressParts(rich bool, p datamodel.EpicProgress) (bar, label string) {
	if p.Total == 0 {
		return "", ""
	}
	filled := p.Done * progressBarWidth / p.Total
	fill, empty, lbrace, rbrace := "▰", "▱", "", ""
	if !rich {
		fill, empty, lbrace, rbrace = "#", "-", "[", "]"
	}
	bar = lbrace + strings.Repeat(fill, filled) + strings.Repeat(empty, progressBarWidth-filled) + rbrace
	return bar, fmt.Sprintf(" %d/%d", p.Done, p.Total)
}
