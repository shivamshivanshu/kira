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
	want   string
	dirty  bool
	err    error
}

func newDetailHost() detailHost {
	return detailHost{panel: newDetailPanel(), cache: map[string]*datamodel.ShowResult{}}
}

func (h *detailHost) render(t theme.Theme, ic iconSet, width, height int) string {
	if h.err != nil {
		return frameOf(t, width, height).Render(t.Heat.Hot.Render("cannot load: " + firstNonEmptyLine(h.err.Error())))
	}
	return h.panel.render(t, ic, h.detail, width, height)
}

func (h *detailHost) update(m *model, key string) (tea.Cmd, bool) {
	return h.panel.update(m, h.detail, key)
}

func (h *detailHost) resetCache() { h.cache = map[string]*datamodel.ShowResult{} }

func (h *detailHost) sync(m *model, id string) {
	h.err = nil
	if id == "" {
		h.detail, h.dirty = nil, false
		return
	}
	if cached, ok := h.cache[id]; ok {
		h.detail, h.dirty = cached, false
		return
	}
	if m.store == nil {
		h.detail, h.dirty = nil, false
		return
	}
	if m.busy {
		h.want, h.dirty = id, true
		return
	}
	res, err := loadDetail(m.store, m.cfg, id)
	if err != nil {
		h.detail, h.dirty, h.err = nil, false, err
		return
	}
	h.cache[id] = res
	h.detail, h.dirty = res, false
}

func (h *detailHost) settle(m *model) {
	if h.dirty && !m.busy {
		h.dirty = false
		h.sync(m, h.want)
	}
}
