package tui

import "strings"

type KeyBinding struct {
	Key  string
	Desc string
}

func hintLine(keys []KeyBinding) string {
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k.Key + " " + k.Desc
	}
	return strings.Join(parts, "  ")
}

func helpBody(keys []KeyBinding) string {
	lines := make([]string, len(keys))
	for i, k := range keys {
		lines[i] = k.Key + "\t" + k.Desc
	}
	return strings.Join(lines, "\n")
}
