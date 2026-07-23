package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

const detailMinWidth = 30

var detailKeys = []KeyBinding{
	{"j/k", "scroll"},
	{"[/]", "commit"},
	{"enter", "git show"},
	{"gg/G", "top/bottom"},
	{"^d/^u", "half-page"},
}

type detailPanel struct {
	scroller
	commitSel int
	cache     detailContent
}

type detailContent struct {
	res   *datamodel.ShowResult
	width int
	sel   int
	body  string
	lines []string
}

func newDetailPanel() *detailPanel { return &detailPanel{} }

func (d *detailPanel) reset() {
	d.scroll = 0
	d.commitSel = 0
}

func (d *detailPanel) update(m *model, res *datamodel.ShowResult, key string) (tea.Cmd, bool) {
	if d.scroller.update(key, m.mainHeight()/2) {
		return nil, true
	}
	switch key {
	case "[":
		d.commitSel = max(0, d.commitSel-1)
	case "]":
		if res != nil {
			d.commitSel = min(len(res.LinkedCommits)-1, d.commitSel+1)
		}
	case "enter":
		if sha := selectedCommit(res, d.commitSel); sha != "" && m.store != nil {
			return tea.ExecProcess(m.store.CommitShowCmd(sha), func(error) tea.Msg { return nil }), true
		}
	default:
		return nil, false
	}
	return nil, true
}

func selectedCommit(res *datamodel.ShowResult, sel int) string {
	if res == nil || sel < 0 || sel >= len(res.LinkedCommits) {
		return ""
	}
	return res.LinkedCommits[sel].SHA
}

func (d *detailPanel) render(t theme.Theme, ic iconSet, res *datamodel.ShowResult, width, height int) string {
	if res == nil {
		return frameOf(t, width, height).Render(t.Dim.Render("Select an item to preview."))
	}
	if width < detailMinWidth {
		return frameOf(t, width, height).Render(t.Dim.Render("▸ widen"))
	}
	lines := d.contentLines(t, ic, res, width)
	return renderScrollable(t, lines, &d.scroll, width, height)
}

func (d *detailPanel) contentLines(t theme.Theme, ic iconSet, res *datamodel.ShowResult, width int) []string {
	if d.cache.res != res || d.cache.width != width {
		d.cache.res, d.cache.width = res, width
		d.cache.body = renderMarkdown(codec.Description(res.Body), width)
		d.cache.sel = -1
	}
	if d.cache.sel != d.commitSel || d.cache.lines == nil {
		d.cache.sel = d.commitSel
		d.cache.lines = detailLines(t, ic, res, width, d.cache.body, d.commitSel)
	}
	return d.cache.lines
}

func detailLines(t theme.Theme, ic iconSet, res *datamodel.ShowResult, width int, body string, sel int) []string {
	var lines []string
	add := func(ss ...string) {
		for _, s := range ss {
			lines = append(lines, strings.Split(s, "\n")...)
		}
	}
	wrap := func(s string) string { return ansi.Wrap(s, width, "") }
	add(t.Accent.Render(wrap(res.Number + "  " + res.Title)))
	add(wrap(detailMeta(t, ic, res)))
	add("", wrap(body))
	if len(res.Comments) > 0 {
		add("", t.Dim.Render("Comments"))
		for _, c := range res.Comments {
			add(t.Dim.Render(wrap(c.Author+"  "+c.Ts)), wrap(strings.TrimRight(c.Text, "\n")))
		}
	}
	if len(res.LinkedCommits) > 0 {
		add("", t.Dim.Render("Linked commits"))
		for i, c := range res.LinkedCommits {
			add(commitLine(t, c, i == sel, width))
		}
	}
	if len(res.ReferencedBy) > 0 {
		add("", t.Dim.Render("Referenced by"))
		for _, c := range res.ReferencedBy {
			add(commitLine(t, c, false, width))
		}
	}
	if len(res.HistoryTail) > 0 {
		add("", t.Dim.Render("History"))
		for _, ev := range res.HistoryTail {
			add(fitWidth(historyLine(ev), width))
		}
	}
	return lines
}

func detailMeta(t theme.Theme, ic iconSet, res *datamodel.ShowResult) string {
	parts := []string{t.CategoryStyle(datamodel.Category(res.Category)).Render("[" + res.State + "]")}
	if res.Subtype != nil && *res.Subtype != "" {
		parts = append(parts, t.Dim.Render("subtype ")+*res.Subtype)
	}
	if res.Owner != nil {
		parts = append(parts, t.Dim.Render("owner ")+*res.Owner)
	}
	if res.Priority != nil {
		parts = append(parts, priorityHue(t, ic.priorityTier(*res.Priority)).Render(*res.Priority))
	}
	if res.Due != nil && *res.Due != "" {
		if datamodel.IsOverdue(res.Due, res.Category, time.Now()) {
			parts = append(parts, t.Dim.Render("due ")+t.Heat.Hot.Render(*res.Due+" overdue"))
		} else {
			parts = append(parts, t.Dim.Render("due ")+*res.Due)
		}
	}
	if len(res.Labels) > 0 {
		parts = append(parts, t.Dim.Render(strings.Join(res.Labels, " ")))
	}
	return strings.Join(parts, "   ")
}

func commitLine(t theme.Theme, c datamodel.CommitLink, selected bool, width int) string {
	marker, style := "  ", t.Text
	if selected {
		marker, style = "> ", t.Accent
	}
	return style.Render(fitWidth(marker+gitx.ShortSHA(c.SHA)+"  "+c.Subject, width))
}

func historyLine(ev datamodel.HistoryEvent) string {
	return histTime(ev.Ts) + "  " + ev.Field + ": " + histValue(ev.From) + " -> " + histValue(ev.To)
}

func histTime(ts string) string {
	if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
		return parsed.Format("2006-01-02 15:04")
	}
	return ts
}

func histValue(v *string) string {
	if v == nil || *v == "" {
		return "(none)"
	}
	return *v
}
