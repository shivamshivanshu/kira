package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type pane int

const (
	paneTree pane = iota
	paneDetail
)

var treeKeys = []KeyBinding{
	{"j/k", "move"},
	{"l/enter", "drill"},
	{"h", "collapse"},
	{"gp", "parent"},
	{"tab", "pane"},
}

func init() { registerScreen(viewTree, func() screen { return newTreeScreen() }) }

type treeScreen struct {
	tree     treeModel
	host     detailHost
	focus    pane
	pendingG bool
}

func newTreeScreen() *treeScreen {
	return &treeScreen{tree: newTreeModel(), host: newDetailHost()}
}

func (s *treeScreen) keys() []KeyBinding { return treeKeys }

func (s *treeScreen) setData(m *model, data treeData) {
	(&s.tree).load(data.nodes, data.fields, data.progress)
	s.host.resetCache()
	m.jumps.dropMissing(func(id string) bool { _, ok := data.fields[id]; return ok })
	s.syncDetail(m)
}

func (s *treeScreen) update(m *model, key string) tea.Cmd {
	if s.pendingG {
		s.pendingG = false
		if key == "p" {
			s.jumpFrom(m)
			(&s.tree).jumpToParent()
			s.syncDetail(m)
		}
		return nil
	}
	if s.focus == paneDetail {
		switch key {
		case "j", "down", "k", "up", "[", "]", "enter":
			return s.host.update(m, key)
		}
	}
	switch key {
	case "j", "down":
		(&s.tree).move(1, m.mainHeight())
		s.syncDetail(m)
	case "k", "up":
		(&s.tree).move(-1, m.mainHeight())
		s.syncDetail(m)
	case "tab", "shift+tab":
		s.toggleFocus()
	case "l", "enter":
		if s.tree.isCollapsedEpic() {
			(&s.tree).setCollapsed(false)
		} else {
			s.jumpFrom(m)
			s.focus = paneDetail
		}
	case "h":
		if !s.tree.isCollapsedEpic() && s.tree.isEpicRow() {
			(&s.tree).setCollapsed(true)
		} else {
			s.jumpFrom(m)
			(&s.tree).jumpToParent()
			s.syncDetail(m)
		}
	case "g":
		s.pendingG = true
	}
	return nil
}

func (s *treeScreen) back(m *model) bool {
	if s.focus == paneDetail {
		s.focus = paneTree
		return true
	}
	return false
}

func (s *treeScreen) focusItem(m *model, id string) {
	for i, r := range s.tree.rows {
		if r.node.ID == id {
			s.tree.cursor = i
			break
		}
	}
	s.syncDetail(m)
}

func (s *treeScreen) view(m *model, width, height int) string {
	if !splitDetail(width) {
		if s.focus == paneDetail {
			return s.host.render(m.theme, width, height)
		}
		return s.tree.render(m.theme, m.icons, width, height, true, false)
	}
	return splitPane(m.theme, width, height,
		func(w int) string { return s.tree.render(m.theme, m.icons, w, height, s.focus == paneTree, true) },
		func(w int) string { return s.host.render(m.theme, w, height) })
}

func (s *treeScreen) toggleFocus() {
	if s.focus == paneTree {
		s.focus = paneDetail
	} else {
		s.focus = paneTree
	}
}

func (s *treeScreen) jumpFrom(m *model) {
	m.jumps.push(jumpEntry{view: m.view, itemID: s.tree.selectedID()})
}

func (s *treeScreen) syncDetail(m *model) {
	s.host.panel.reset()
	s.host.sync(m, s.tree.selectedID())
}
