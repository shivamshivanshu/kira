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
	fill, empty, open, close := "▰", "▱", "", ""
	if !rich {
		fill, empty, open, close = "#", "-", "[", "]"
	}
	bar = open + strings.Repeat(fill, filled) + strings.Repeat(empty, progressBarWidth-filled) + close
	return bar, fmt.Sprintf(" %d/%d", p.Done, p.Total)
}
