package tui

import (
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
)

var (
	mdMu    sync.Mutex
	mdCache = map[int]*glamour.TermRenderer{}
)

func renderMarkdown(body string, width int) string {
	body = strings.TrimRight(body, "\n")
	if width <= 0 || body == "" {
		return body
	}
	r := markdownRenderer(width)
	if r == nil {
		return body
	}
	out, err := r.Render(body)
	if err != nil {
		return body
	}
	return strings.Trim(out, "\n")
}

func markdownRenderer(width int) *glamour.TermRenderer {
	mdMu.Lock()
	defer mdMu.Unlock()
	if r, ok := mdCache[width]; ok {
		return r
	}
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle("notty"), glamour.WithWordWrap(width))
	if err != nil {
		return nil
	}
	mdCache[width] = r
	return r
}
