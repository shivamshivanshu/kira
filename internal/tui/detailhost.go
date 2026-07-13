package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

type detailHost struct {
	panel  *detailPanel
	detail *datamodel.ShowResult
	cache  map[string]*datamodel.ShowResult
}

func newDetailHost() detailHost {
	return detailHost{panel: newDetailPanel(), cache: map[string]*datamodel.ShowResult{}}
}

func (h *detailHost) render(t theme.Theme, width, height int) string {
	return h.panel.render(t, h.detail, width, height)
}

func (h *detailHost) update(m *model, key string) tea.Cmd {
	return h.panel.update(m, h.detail, key)
}

func (h *detailHost) resetCache() { h.cache = map[string]*datamodel.ShowResult{} }

func (h *detailHost) sync(m *model, id string) {
	if id == "" {
		h.detail = nil
		return
	}
	if cached, ok := h.cache[id]; ok {
		h.detail = cached
		return
	}
	if m.store == nil {
		h.detail = nil
		return
	}
	res, err := loadDetail(m.store, m.cfg, id)
	if err != nil {
		h.detail = nil
		return
	}
	h.cache[id] = res
	h.detail = res
}
