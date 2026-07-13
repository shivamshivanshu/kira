package tui

import (
	"math"
	"strings"
)

var (
	sparkRich  = []rune("▁▂▃▄▅▆▇█")
	sparkAscii = []rune(".:-=+*#@")
)

func sparkline(vals []float64, rich bool) string {
	if len(vals) == 0 {
		return ""
	}
	ramp := sparkAscii
	if rich {
		ramp = sparkRich
	}
	var maxV float64
	for _, v := range vals {
		maxV = max(maxV, v)
	}
	var b strings.Builder
	for _, v := range vals {
		b.WriteRune(ramp[sparkLevel(v, maxV, len(ramp))])
	}
	return b.String()
}

func sparkLevel(v, maxV float64, levels int) int {
	if maxV <= 0 || v <= 0 {
		return 0
	}
	return clamp(int(math.Round(v/maxV*float64(levels-1))), 0, levels-1)
}
