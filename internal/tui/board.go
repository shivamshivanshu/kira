package tui

import (
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/termx"
	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

type boardModel struct {
	result *datamodel.BoardResult
	col    int
	row    int
}

func (bm *boardModel) load(res *datamodel.BoardResult) {
	bm.result = res
	bm.clamp()
}

func (bm *boardModel) columns() []datamodel.BoardColumn {
	if bm.result == nil {
		return nil
	}
	return bm.result.Columns
}

func (bm *boardModel) colLen() int {
	cols := bm.columns()
	if bm.col < 0 || bm.col >= len(cols) {
		return 0
	}
	return len(cols[bm.col].Items)
}

func (bm *boardModel) clamp() {
	bm.col = clamp(bm.col, 0, len(bm.columns())-1)
	bm.row = clamp(bm.row, 0, bm.colLen()-1)
}

func (bm *boardModel) moveCol(d int) {
	bm.col += d
	bm.clamp()
}

func (bm *boardModel) moveRow(d int) { bm.row = clamp(bm.row+d, 0, bm.colLen()-1) }

func (bm *boardModel) toTop() { bm.row = 0 }

func (bm *boardModel) toBottom() { bm.row = max(0, bm.colLen()-1) }

func (bm *boardModel) selected() (datamodel.ListItem, bool) {
	if bm.colLen() == 0 {
		return datamodel.ListItem{}, false
	}
	return bm.columns()[bm.col].Items[bm.row], true
}

func (bm *boardModel) focusByID(id string) {
	for ci, c := range bm.columns() {
		for ri, it := range c.Items {
			if it.ID == id {
				bm.col, bm.row = ci, ri
				return
			}
		}
	}
}

func renderBoard(t theme.Theme, ic iconSet, res *datamodel.BoardResult, width, height, focusCol, focusRow int) string {
	if res == nil || res.Empty() {
		return renderBoardEmpty(t, res, width, height)
	}
	n := len(res.Columns)
	widths := splitWidth(width-(n-1), n)
	sep := verticalRule(t.Border.Render("│"), height)
	blocks := make([]string, 0, 2*n-1)
	for i, col := range res.Columns {
		fr := -1
		if i == focusCol {
			fr = focusRow
		}
		blocks = append(blocks, renderColumn(t, ic, col, widths[i], height, i == focusCol, fr))
		if i < n-1 {
			blocks = append(blocks, sep)
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, blocks...)
}

func renderColumn(t theme.Theme, ic iconSet, col datamodel.BoardColumn, w, height int, focused bool, focusRow int) string {
	fit := t.Renderer().NewStyle().Width(w).MaxWidth(w)
	lines := make([]string, 0, height)
	lines = append(lines, fit.Render(columnHeaderStyle(t, col, focused).Render(fitWidth(columnLabel(col), w))))
	if height > 1 {
		lines = append(lines, fit.Render(t.Dim.Render(strings.Repeat("─", max(0, w)))))
	}

	cat := datamodel.Category(col.Category)
	capacity := height - len(lines)
	start, cardSlots, above, below := cardWindow(len(col.Items), capacity, focusRow)
	if above > 0 {
		lines = append(lines, fit.Render(t.Dim.Render("+"+strconv.Itoa(above)+" above")))
	}
	for i := start; i < start+cardSlots; i++ {
		lines = append(lines, fit.Render(renderCard(t, ic, cat, col.Items[i], w, focused && i == focusRow)))
	}
	if below > 0 {
		lines = append(lines, fit.Render(t.Dim.Render("+"+strconv.Itoa(below)+" more")))
	}
	for len(lines) < height {
		lines = append(lines, fit.Render(""))
	}
	return strings.Join(lines, "\n")
}

func renderCard(t theme.Theme, ic iconSet, cat datamodel.Category, it datamodel.ListItem, w int, selected bool) string {
	priority := deref(it.Priority)
	glyph := ic.categoryGlyph(cat, it.Resolution)
	prio := ic.priorityCell(priority)
	segments := []rowSegment{
		{prio, priorityHue(t, ic.priorityTier(priority))}, {" ", t.Text},
		{glyph, t.CategoryStyle(cat)}, {" ", t.Text},
	}
	if overdue(it.Due, it.Category) {
		segments = append(segments, rowSegment{ic.overdueGlyph(), t.Heat.Hot}, rowSegment{" ", t.Text})
	}
	segments = append(segments, rowSegment{it.Number, t.Dim}, rowSegment{"  ", t.Text})
	title := fitWidth(it.Title, w-2-segmentsWidth(segments))
	segments = append(segments, rowSegment{title, t.Text})
	lead := " "
	if selected {
		lead = t.Accent.Render("▌")
	}
	return lead + renderSegments(segments, selected)
}

func cardWindow(total, capacity, focusRow int) (start, slots, above, below int) {
	if capacity <= 0 || total <= 0 {
		return 0, 0, 0, 0
	}
	if total <= capacity {
		return 0, total, 0, 0
	}
	if capacity <= 2 {
		slots = capacity
		start = clamp(focusRow-slots+1, 0, total-slots)
		return start, slots, 0, 0
	}
	slots = capacity - 1
	if focusRow < slots {
		return 0, slots, 0, total - slots
	}
	start = total - slots
	if focusRow >= start {
		return start, slots, start, 0
	}
	slots = capacity - 2
	start = focusRow - slots + 1
	return start, slots, start, total - start - slots
}

func columnLabel(col datamodel.BoardColumn) string {
	if col.Wip > 0 {
		return col.State + "  " + strconv.Itoa(col.Count) + "/" + strconv.Itoa(col.Wip)
	}
	return col.State
}

func columnHeaderStyle(t theme.Theme, col datamodel.BoardColumn, focused bool) lipgloss.Style {
	heat := func(s lipgloss.Style) lipgloss.Style {
		if focused {
			return s.Bold(true)
		}
		return s
	}
	switch {
	case col.Wip > 0 && col.Count > col.Wip:
		return heat(t.Heat.Hot)
	case col.Wip > 0 && col.Count == col.Wip:
		return heat(t.Heat.Warm)
	case focused:
		return heat(t.Accent)
	default:
		return t.Dim
	}
}

const boardEmptyMessage = "No tickets on the board yet."

func renderBoardEmpty(t theme.Theme, res *datamodel.BoardResult, width, height int) string {
	var b strings.Builder
	b.WriteString(t.Dim.Render(boardEmptyMessage))
	if res != nil && len(res.Columns) > 0 {
		names := make([]string, len(res.Columns))
		for i, c := range res.Columns {
			names[i] = c.State
		}
		b.WriteString("\n\n")
		b.WriteString(t.Dim.Render("Columns: " + strings.Join(names, "  ·  ")))
	}
	b.WriteString("\n\n")
	b.WriteString(t.Dim.Render("Create a ticket — n create · : command"))
	return centered(t, width, height, b.String())
}

func verticalRule(cell string, height int) string {
	lines := make([]string, max(0, height))
	for i := range lines {
		lines[i] = cell
	}
	return strings.Join(lines, "\n")
}

func splitWidth(total, n int) []int {
	if n <= 0 {
		return nil
	}
	if total < n {
		total = n
	}
	base, rem := total/n, total%n
	widths := make([]int, n)
	for i := range widths {
		widths[i] = base
		if i < rem {
			widths[i]++
		}
	}
	return widths
}

func RenderBoardPlain(w io.Writer, cfg *datamodel.Config, res *datamodel.BoardResult, width int, noColor bool) error {
	th := theme.For(w, cfg.UI, noColor)
	out := renderBoard(th, detectIcons(cfg.UI.Icons, cfg.Priorities.Values, cfg.ResolutionsDropped, osEnv, termx.WriterIsTTY(w)), res, width, plainHeight(res), -1, -1)
	_, err := io.WriteString(w, out+"\n")
	return err
}

func plainHeight(res *datamodel.BoardResult) int {
	if res == nil || res.Empty() {
		return 6
	}
	maxCards := 0
	for _, c := range res.Columns {
		if len(c.Items) > maxCards {
			maxCards = len(c.Items)
		}
	}
	return maxCards + 2
}
