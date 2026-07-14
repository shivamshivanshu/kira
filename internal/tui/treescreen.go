package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/showfmt"
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
	{"gg/G", "top/bottom"},
	{"^d/^u", "half-page"},
	{"zM/zR", "collapse/expand"},
	{"e", "edit"},
	{"v", "view"},
}

type treeScreen struct {
	tree  treeModel
	host  detailHost
	focus pane
	chord chord
}

func newTreeScreen() *treeScreen {
	return &treeScreen{tree: newTreeModel(), host: newDetailHost()}
}

func (s *treeScreen) keys() []KeyBinding { return treeKeys }

func (s *treeScreen) invalidate() {}

func (s *treeScreen) setData(m *model, data treeData) {
	s.tree.load(data.nodes, data.fields, data.progress)
	s.host.resetCache()
	m.jumps.dropMissing(func(id string) bool { _, ok := data.fields[id]; return ok })
	s.syncDetail(m)
}

func (s *treeScreen) update(m *model, key string) tea.Cmd {
	if p, ok := s.chord.take(); ok {
		switch p + key {
		case "gp":
			s.jumpFrom(m)
			s.tree.jumpToParent()
			s.syncDetail(m)
		case "gg":
			s.tree.toTop(m.mainHeight())
			s.syncDetail(m)
		case "zM":
			s.tree.collapseAll()
			s.syncDetail(m)
		case "zR":
			s.tree.expandAll()
			s.syncDetail(m)
		}
		return nil
	}
	if s.focus == paneDetail {
		if cmd, handled := s.host.update(m, key); handled {
			return cmd
		}
	}
	switch key {
	case "j", "down":
		s.tree.move(1, m.mainHeight())
		s.syncDetail(m)
	case "k", "up":
		s.tree.move(-1, m.mainHeight())
		s.syncDetail(m)
	case "tab", "shift+tab":
		s.toggleFocus()
	case "l", "enter":
		if s.tree.isCollapsedEpic() {
			s.tree.setCollapsed(false)
		} else {
			s.jumpFrom(m)
			s.focus = paneDetail
		}
	case "h":
		if !s.tree.isCollapsedEpic() && s.tree.isEpicRow() {
			s.tree.setCollapsed(true)
		} else {
			s.jumpFrom(m)
			s.tree.jumpToParent()
			s.syncDetail(m)
		}
	case "g", "z":
		s.chord.arm(key)
	case "G":
		s.tree.toBottom(m.mainHeight())
		s.syncDetail(m)
	case "ctrl+d":
		s.tree.move(m.mainHeight()/2, m.mainHeight())
		s.syncDetail(m)
	case "ctrl+u":
		s.tree.move(-m.mainHeight()/2, m.mainHeight())
		s.syncDetail(m)
	case "e":
		return s.editSelected(m)
	case "v":
		return s.viewSelected(m)
	}
	return nil
}

func (s *treeScreen) editSelected(m *model) tea.Cmd {
	return s.suspendForSelected(m, editItemCmd)
}

func (s *treeScreen) viewSelected(m *model) tea.Cmd {
	return s.suspendForSelected(m, viewItemCmd)
}

func (s *treeScreen) suspendForSelected(m *model, open func(*core.Store, *datamodel.Config, string) (tea.Cmd, error)) tea.Cmd {
	id := s.tree.selectedID()
	if id == "" || m.store == nil {
		return nil
	}
	cmd, err := open(m.store, m.cfg, id)
	if err != nil {
		m.bar.setError(firstNonEmptyLine(err.Error()))
		return nil
	}
	return cmd
}

func (s *treeScreen) back(m *model) bool {
	if s.focus == paneDetail {
		s.focus = paneTree
		return true
	}
	return false
}

func (s *treeScreen) focusItem(m *model, id string) {
	s.tree.focusID(id)
	s.syncDetail(m)
}

func (s *treeScreen) focusedItem() (showfmt.Item, bool) {
	row := s.tree.current()
	if row == nil {
		return showfmt.Item{}, false
	}
	return showfmt.Item{ID: row.node.ID, Number: row.node.Number, Title: row.node.Title}, true
}

func (s *treeScreen) view(m *model, width, height int) string {
	if !splitDetail(width) {
		if s.focus == paneDetail {
			return s.host.render(m.theme, m.icons, width, height)
		}
		return s.tree.render(m.theme, m.icons, width, height, true, false)
	}
	return splitPane(m.theme, m.cfg.UI.Tui.Split, width, height,
		func(w int) string { return s.tree.render(m.theme, m.icons, w, height, s.focus == paneTree, true) },
		func(w int) string { return s.host.render(m.theme, m.icons, w, height) })
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

func (s *treeScreen) settle(m *model) { s.host.settle(m) }
