package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/shivamshivanshu/kira/internal/datamodel"
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
	tree        treeModel
	detail      *datamodel.ShowResult
	detailCache map[string]*datamodel.ShowResult
	panel       *detailPanel
	focus       pane
	pendingG    bool
}

func newTreeScreen() *treeScreen {
	return &treeScreen{tree: newTreeModel(), detailCache: map[string]*datamodel.ShowResult{}, panel: newDetailPanel()}
}

func (s *treeScreen) keys() []KeyBinding { return treeKeys }

func (s *treeScreen) apply(m *model, data treeData) {
	(&s.tree).load(data.nodes, data.fields, data.progress)
	s.detailCache = map[string]*datamodel.ShowResult{}
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
			return s.panel.update(m, s.detail, key)
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
			return s.panel.render(m.theme, s.detail, width, height)
		}
		return s.tree.render(m.theme, m.icons, width, height, true, false)
	}
	tw := treeWidth(width)
	left := s.tree.render(m.theme, m.icons, tw, height, s.focus == paneTree, true)
	sep := strings.TrimRight(strings.Repeat(m.theme.Border.Render("│")+"\n", height), "\n")
	right := s.panel.render(m.theme, s.detail, width-tw-1, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
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
	s.panel.reset()
	id := s.tree.selectedID()
	if id == "" {
		s.detail = nil
		return
	}
	if cached, ok := s.detailCache[id]; ok {
		s.detail = cached
		return
	}
	if m.store == nil {
		s.detail = nil
		return
	}
	res, err := loadDetail(m.store, m.cfg, id)
	if err != nil {
		s.detail = nil
		return
	}
	s.detailCache[id] = res
	s.detail = res
}
