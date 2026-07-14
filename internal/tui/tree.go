package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

const treeEmptyMessage = "No tickets yet — n create · : command · ? keys"

type treeItem struct {
	node   datamodel.TreeNode
	fields datamodel.ListItem
	depth  int
}

func (ti treeItem) isEpic() bool { return len(ti.node.Children) > 0 }

type treeModel struct {
	nodes     []datamodel.TreeNode
	fields    map[string]datamodel.ListItem
	progress  map[string]datamodel.EpicProgress
	collapsed map[string]bool

	rows   []treeItem
	cursor int
	top    int
}

func newTreeModel() treeModel {
	return treeModel{fields: map[string]datamodel.ListItem{}, progress: map[string]datamodel.EpicProgress{}, collapsed: map[string]bool{}}
}

func (tm *treeModel) load(nodes []datamodel.TreeNode, fields map[string]datamodel.ListItem, progress map[string]datamodel.EpicProgress) {
	tm.nodes = nodes
	tm.fields = fields
	tm.progress = progress
	tm.flatten()
}

func (tm *treeModel) flatten() {
	tm.rows = tm.rows[:0]
	var walk func(n datamodel.TreeNode, depth int)
	walk = func(n datamodel.TreeNode, depth int) {
		tm.rows = append(tm.rows, treeItem{node: n, depth: depth, fields: tm.fields[n.ID]})
		if len(n.Children) > 0 && !tm.collapsed[n.ID] {
			for _, c := range n.Children {
				walk(c, depth+1)
			}
		}
	}
	for _, n := range tm.nodes {
		walk(n, 0)
	}
	tm.cursor = clamp(tm.cursor, 0, len(tm.rows)-1)
}

func (tm *treeModel) empty() bool { return len(tm.rows) == 0 }

func (tm *treeModel) current() *treeItem {
	if tm.cursor < 0 || tm.cursor >= len(tm.rows) {
		return nil
	}
	return &tm.rows[tm.cursor]
}

func (tm *treeModel) move(delta, visible int) {
	tm.cursor = clamp(tm.cursor+delta, 0, len(tm.rows)-1)
	tm.scrollInto(visible)
}

func (tm *treeModel) toTop(visible int) {
	tm.cursor = 0
	tm.scrollInto(visible)
}

func (tm *treeModel) toBottom(visible int) {
	tm.cursor = max(0, len(tm.rows)-1)
	tm.scrollInto(visible)
}

func (tm *treeModel) scrollInto(visible int) {
	if visible <= 0 {
		return
	}
	if tm.cursor < tm.top {
		tm.top = tm.cursor
	}
	if tm.cursor >= tm.top+visible {
		tm.top = tm.cursor - visible + 1
	}
	tm.top = clamp(tm.top, 0, max(0, len(tm.rows)-1))
}

func (tm *treeModel) setCollapsed(v bool) {
	cur := tm.current()
	if cur == nil || !cur.isEpic() {
		return
	}
	if v {
		tm.collapsed[cur.node.ID] = true
	} else {
		delete(tm.collapsed, cur.node.ID)
	}
	tm.flatten()
}

func (tm *treeModel) isEpicRow() bool {
	cur := tm.current()
	return cur != nil && cur.isEpic()
}

func (tm *treeModel) isCollapsedEpic() bool {
	cur := tm.current()
	return cur != nil && cur.isEpic() && tm.collapsed[cur.node.ID]
}

func (tm *treeModel) collapseAll() {
	target := tm.rootAncestor(tm.selectedID())
	var walk func(nodes []datamodel.TreeNode)
	walk = func(nodes []datamodel.TreeNode) {
		for _, n := range nodes {
			if len(n.Children) > 0 {
				tm.collapsed[n.ID] = true
				walk(n.Children)
			}
		}
	}
	walk(tm.nodes)
	tm.flatten()
	tm.focusID(target)
}

func (tm *treeModel) expandAll() {
	target := tm.selectedID()
	tm.collapsed = map[string]bool{}
	tm.flatten()
	tm.focusID(target)
}

func (tm *treeModel) rootAncestor(id string) string {
	for {
		f, ok := tm.fields[id]
		if !ok || f.Epic == nil || *f.Epic == "" {
			return id
		}
		id = *f.Epic
	}
}

func (tm *treeModel) jumpToParent() {
	if cur := tm.current(); cur != nil && cur.fields.Epic != nil {
		tm.focusID(*cur.fields.Epic)
	}
}

func (tm *treeModel) focusID(id string) {
	for i, r := range tm.rows {
		if r.node.ID == id {
			tm.cursor = i
			return
		}
	}
}

func (tm *treeModel) selectedID() string {
	if cur := tm.current(); cur != nil {
		return cur.node.ID
	}
	return ""
}

func (tm *treeModel) render(t theme.Theme, ic iconSet, width, height int, focused, showProgress bool) string {
	if tm.empty() {
		msg := t.Dim.Render(treeEmptyMessage)
		return t.Renderer().NewStyle().Width(width).Height(height).Align(lipgloss.Center, lipgloss.Center).Render(msg)
	}
	tm.scrollInto(height)
	lines := make([]string, 0, height)
	for i := tm.top; i < len(tm.rows) && i < tm.top+height; i++ {
		lines = append(lines, tm.renderRow(t, ic, tm.rows[i], width, focused && i == tm.cursor, showProgress))
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

type rowSegment struct {
	text  string
	style lipgloss.Style
}

func (tm *treeModel) renderRow(t theme.Theme, ic iconSet, ti treeItem, width int, selected, showProgress bool) string {
	marker := " "
	if ti.isEpic() {
		if tm.collapsed[ti.node.ID] {
			marker = "▸"
		} else {
			marker = "▾"
		}
	}
	cat := datamodel.Category(ti.fields.Category)
	indent := strings.Repeat("  ", ti.depth)
	priority := deref(ti.fields.Priority)

	fixed := []rowSegment{
		{marker, t.Dim}, {" ", t.Text}, {indent, t.Text},
		{ic.typeGlyph(ti.node.Type), t.Dim}, {" ", t.Text}, {ti.node.Number, t.Dim}, {"  ", t.Text},
	}
	right := []rowSegment{
		{" ", t.Text}, {ic.priorityCell(priority), priorityHue(t, ic.priorityTier(priority))},
		{" ", t.Text}, {ic.categoryGlyph(cat, ti.fields.Resolution), t.CategoryStyle(cat)},
		{" ", t.Text}, {"[" + ti.fields.State + "]", t.CategoryStyle(cat)},
	}
	if overdue(ti.fields.Due, ti.fields.Category) {
		right = append(right, rowSegment{" ", t.Text}, rowSegment{ic.overdueGlyph(), t.Heat.Hot})
	}
	if showProgress && ti.isEpic() {
		if bar, label := progressParts(ic.rich(), tm.progress[ti.node.ID]); bar != "" {
			right = append(right, rowSegment{" ", t.Text}, rowSegment{bar, t.CategoryStyle(datamodel.CategoryDone)}, rowSegment{label, t.Dim})
		}
	}

	budget := width - 1 - segmentsWidth(fixed) - segmentsWidth(right)
	title := fitWidth(ti.node.Title, budget)
	pad := max(0, budget-lipgloss.Width(title))
	segments := append(fixed, rowSegment{title, t.Text}, rowSegment{strings.Repeat(" ", pad), t.Text})
	segments = append(segments, right...)

	fit := t.Renderer().NewStyle().Width(width).MaxWidth(width)
	if selected {
		return fit.Render(t.Accent.Render("▌") + renderSegments(segments, true))
	}
	return fit.Render(" " + renderSegments(segments, false))
}

func segmentsWidth(segs []rowSegment) int {
	w := 0
	for _, s := range segs {
		w += lipgloss.Width(s.text)
	}
	return w
}

func renderSegments(segs []rowSegment, bold bool) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(styleText(s.style, s.text, bold))
	}
	return b.String()
}
